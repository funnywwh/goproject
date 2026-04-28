package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"prjflow/internal/model"
	"prjflow/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BugHandler struct {
	db *gorm.DB
}

func NewBugHandler(db *gorm.DB) *BugHandler {
	return &BugHandler{db: db}
}

func splitQueryValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}

	return result
}

func getStatusFilters(c *gin.Context) []string {
	queryValues := append([]string{}, c.QueryArray("status")...)
	queryValues = append(queryValues, c.QueryArray("status[]")...)

	return splitQueryValues(queryValues)
}

func applyStatusFilter(query *gorm.DB, statuses []string) *gorm.DB {
	switch len(statuses) {
	case 0:
		return query
	case 1:
		return query.Where("status = ?", statuses[0])
	default:
		return query.Where("status IN ?", statuses)
	}
}

func (h *BugHandler) applyBugBaseFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	// 搜索
	if keyword := c.Query("keyword"); keyword != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	// 项目筛选
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}

	// 状态筛选
	query = applyStatusFilter(query, getStatusFilters(c))

	// 优先级筛选
	if priority := c.Query("priority"); priority != "" {
		query = query.Where("priority = ?", priority)
	}

	// 严重程度筛选
	if severity := c.Query("severity"); severity != "" {
		query = query.Where("severity = ?", severity)
	}

	// 需求筛选
	if requirementID := c.Query("requirement_id"); requirementID != "" {
		query = query.Where("requirement_id = ?", requirementID)
	}

	// 功能模块筛选
	if moduleID := c.Query("module_id"); moduleID != "" {
		query = query.Where("module_id = ?", moduleID)
	}

	// 创建人筛选
	if creatorID := c.Query("creator_id"); creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}

	// 更新日期范围筛选
	if startDate := c.Query("updated_start_date"); startDate != "" {
		if startTime, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("updated_at >= ?", startTime)
		}
	}
	if endDate := c.Query("updated_end_date"); endDate != "" {
		if endTime, err := time.Parse("2006-01-02", endDate); err == nil {
			// 结束日期包含整天，所以加一天
			endTime = endTime.AddDate(0, 0, 1)
			query = query.Where("updated_at < ?", endTime)
		}
	}

	return query
}

// GetBugs 获取Bug列表
func (h *BugHandler) GetBugs(c *gin.Context) {
	var bugs []model.Bug
	query := h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("Versions")

	// 权限过滤：普通用户只能看到自己创建或参与的Bug
	query = utils.FilterBugsByUser(h.db, c, query)
	query = h.applyBugBaseFilters(c, query)

	// 版本筛选（通过关联表）
	hasVersionFilter := false
	var versionBugIDs []uint
	if versionID := c.Query("version_id"); versionID != "" {
		// 当有版本筛选时，先查询符合条件的 Bug ID
		idQuery := h.db.Model(&model.Bug{}).Select("DISTINCT bugs.id")

		// 应用权限过滤
		idQuery = utils.FilterBugsByUser(h.db, c, idQuery)
		idQuery = h.applyBugBaseFilters(c, idQuery)

		// JOIN 版本关联表
		idQuery = idQuery.Joins("JOIN version_bugs ON version_bugs.bug_id = bugs.id").
			Where("version_bugs.version_id = ?", versionID)

		// 获取符合条件的 Bug ID 列表
		if err := idQuery.Pluck("bugs.id", &versionBugIDs).Error; err != nil {
			utils.Error(c, utils.CodeError, "查询失败")
			return
		}

		// 如果没有符合条件的 Bug，直接返回空结果
		if len(versionBugIDs) == 0 {
			utils.Success(c, gin.H{
				"list":      []model.Bug{},
				"total":     0,
				"page":      utils.GetPage(c),
				"page_size": utils.GetPageSize(c),
			})
			return
		}

		hasVersionFilter = true
	}

	// 分配人筛选（通过关联表）
	hasAssigneeFilter := false
	var bugIDs []uint
	if assigneeID := c.Query("assignee_id"); assigneeID != "" {
		// 当有分配人筛选时，先查询符合条件的 Bug ID
		idQuery := h.db.Model(&model.Bug{}).Select("DISTINCT bugs.id")

		// 应用权限过滤
		idQuery = utils.FilterBugsByUser(h.db, c, idQuery)
		idQuery = h.applyBugBaseFilters(c, idQuery)

		// JOIN 分配人表
		idQuery = idQuery.Joins("JOIN bug_assignees ON bug_assignees.bug_id = bugs.id").
			Where("bug_assignees.user_id = ?", assigneeID)

		// 获取符合条件的 Bug ID 列表
		if err := idQuery.Pluck("bugs.id", &bugIDs).Error; err != nil {
			utils.Error(c, utils.CodeError, "查询失败")
			return
		}

		// 如果没有符合条件的 Bug，直接返回空结果
		if len(bugIDs) == 0 {
			utils.Success(c, gin.H{
				"list":      []model.Bug{},
				"total":     0,
				"page":      utils.GetPage(c),
				"page_size": utils.GetPageSize(c),
			})
			return
		}

		// 使用 ID 列表查询，避免 JOIN + GROUP BY 对 Preload 的影响
		query = query.Where("bugs.id IN ?", bugIDs)
		hasAssigneeFilter = true
	}

	// 如果有版本筛选，需要合并到查询条件中
	if hasVersionFilter {
		if hasAssigneeFilter {
			// 如果同时有分配人和版本筛选，取交集
			var intersectionIDs []uint
			for _, id := range bugIDs {
				for _, vid := range versionBugIDs {
					if id == vid {
						intersectionIDs = append(intersectionIDs, id)
						break
					}
				}
			}
			if len(intersectionIDs) == 0 {
				utils.Success(c, gin.H{
					"list":      []model.Bug{},
					"total":     0,
					"page":      utils.GetPage(c),
					"page_size": utils.GetPageSize(c),
				})
				return
			}
			query = query.Where("bugs.id IN ?", intersectionIDs)
			bugIDs = intersectionIDs
		} else {
			query = query.Where("bugs.id IN ?", versionBugIDs)
			bugIDs = versionBugIDs
		}
		hasAssigneeFilter = true // 标记为已过滤，以便在countQuery中使用
	}

	// 分页
	page := utils.GetPage(c)
	pageSize := utils.GetPageSize(c)
	offset := (page - 1) * pageSize

	// 构建 countQuery，应用与 query 相同的筛选条件
	var total int64
	countQuery := utils.FilterBugsByUser(h.db, c, h.db.Model(&model.Bug{}))
	countQuery = h.applyBugBaseFilters(c, countQuery)

	// 分配人和版本筛选（通过关联表）
	if hasAssigneeFilter {
		// 使用已查询的 ID 列表进行计数
		countQuery = countQuery.Where("id IN ?", bugIDs)
	}

	// 计数
	countQuery.Count(&total)

	// 查询数据（现在 query 已经应用了所有筛选条件，包括 ID 列表）
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&bugs).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询失败")
		return
	}

	utils.Success(c, gin.H{
		"list":      bugs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetBug 获取Bug详情
func (h *BugHandler) GetBug(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug

	if err := h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("Versions").Preload("Attachments").Preload("Attachments.Creator").First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能查看自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限访问该Bug")
		return
	}

	utils.Success(c, bug)
}

// CreateBug 创建Bug
func (h *BugHandler) CreateBug(c *gin.Context) {
	var req struct {
		Title          string   `json:"title" binding:"required"`
		Description    string   `json:"description"`
		Status         string   `json:"status"`
		Priority       string   `json:"priority"`
		Severity       string   `json:"severity"`
		ProjectID      uint     `json:"project_id" binding:"required"`
		RequirementID  *uint    `json:"requirement_id"`
		ModuleID       *uint    `json:"module_id"`
		AssigneeIDs    []uint   `json:"assignee_ids"`
		EstimatedHours *float64 `json:"estimated_hours"`
		VersionIDs     []uint   `json:"version_ids" binding:"required,min=1"` // 所属版本ID列表（必填，至少一个）
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未登录")
		return
	}

	// 验证状态
	if req.Status == "" {
		req.Status = "active"
	}
	validStatuses := map[string]bool{
		"active":   true,
		"resolved": true,
		"closed":   true,
	}
	if !validStatuses[req.Status] {
		utils.Error(c, 400, "状态值无效，有效值：active, resolved, closed")
		return
	}

	// 验证优先级
	if req.Priority == "" {
		req.Priority = "medium"
	}
	validPriorities := map[string]bool{
		"low":    true,
		"medium": true,
		"high":   true,
		"urgent": true,
	}
	if !validPriorities[req.Priority] {
		utils.Error(c, 400, "优先级值无效")
		return
	}

	// 验证严重程度
	if req.Severity == "" {
		req.Severity = "medium"
	}
	validSeverities := map[string]bool{
		"low":      true,
		"medium":   true,
		"high":     true,
		"critical": true,
	}
	if !validSeverities[req.Severity] {
		utils.Error(c, 400, "严重程度值无效")
		return
	}

	// 验证项目是否存在
	var project model.Project
	if err := h.db.First(&project, req.ProjectID).Error; err != nil {
		utils.Error(c, 400, "项目不存在")
		return
	}

	// 权限检查：普通用户只能在自己参与的项目中创建Bug
	if !utils.CheckProjectAccess(h.db, c, project.ID) {
		utils.Error(c, 403, "没有权限在该项目中创建Bug")
		return
	}

	// 如果指定了需求，验证需求是否存在
	if req.RequirementID != nil {
		var requirement model.Requirement
		if err := h.db.First(&requirement, *req.RequirementID).Error; err != nil {
			utils.Error(c, 400, "需求不存在")
			return
		}
	}

	// 如果指定了功能模块，验证模块是否存在
	if req.ModuleID != nil {
		var module model.Module
		if err := h.db.First(&module, *req.ModuleID).Error; err != nil {
			utils.Error(c, 400, "功能模块不存在")
			return
		}
	}

	// 验证分配人是否存在
	if len(req.AssigneeIDs) > 0 {
		var users []model.User
		if err := h.db.Where("id IN ?", req.AssigneeIDs).Find(&users).Error; err != nil || len(users) != len(req.AssigneeIDs) {
			utils.Error(c, 400, "分配人不存在")
			return
		}
	}

	// 验证版本是否存在且属于同一项目（必填，至少一个）
	if len(req.VersionIDs) == 0 {
		utils.Error(c, 400, "必须至少选择一个所属版本")
		return
	}
	var versions []model.Version
	if err := h.db.Where("id IN ? AND project_id = ?", req.VersionIDs, req.ProjectID).Find(&versions).Error; err != nil {
		utils.Error(c, 400, "版本查询失败")
		return
	}
	if len(versions) != len(req.VersionIDs) {
		utils.Error(c, 400, "版本不存在或不属于当前项目")
		return
	}

	bug := model.Bug{
		Title:          req.Title,
		Description:    req.Description,
		Status:         req.Status,
		Priority:       req.Priority,
		Severity:       req.Severity,
		ProjectID:      req.ProjectID,
		RequirementID:  req.RequirementID,
		ModuleID:       req.ModuleID,
		CreatorID:      userID.(uint),
		EstimatedHours: req.EstimatedHours,
	}

	if err := h.db.Create(&bug).Error; err != nil {
		utils.Error(c, utils.CodeError, "创建失败")
		return
	}

	// 分配Bug给用户
	if len(req.AssigneeIDs) > 0 {
		var assignees []model.User
		h.db.Where("id IN ?", req.AssigneeIDs).Find(&assignees)
		if err := h.db.Model(&bug).Association("Assignees").Replace(assignees); err != nil {
			utils.Error(c, utils.CodeError, "分配失败")
			return
		}
		// 如果有分配人，状态保持为active（禅道中Bug只有active/resolved/closed三种状态）
		// 不需要改变状态
	}

	// 关联版本到Bug（多对多关系）
	if err := h.db.Model(&bug).Association("Versions").Replace(versions); err != nil {
		utils.Error(c, utils.CodeError, "关联版本失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("ResolvedVersion").Preload("Versions").First(&bug, bug.ID)

	// 记录创建操作
	dbValue, _ := c.Get("db")
	if db, ok := dbValue.(*gorm.DB); ok {
		actionID, _ := utils.RecordAction(db, "bug", bug.ID, "created", userID.(uint), "", nil)
		// 如果创建时就有分配人，记录到历史记录中
		if len(bug.Assignees) > 0 {
			var assigneeIDs []uint
			for _, assignee := range bug.Assignees {
				assigneeIDs = append(assigneeIDs, assignee.ID)
			}
			assigneeIDsStr := formatUintSlice(assigneeIDs)
			if assigneeIDsStr != "" {
				changes := []utils.HistoryChange{
					{Field: "assignee_ids", Old: "", New: assigneeIDsStr},
				}
				utils.RecordHistory(db, actionID, changes)
			}
		}
	}

	utils.Success(c, bug)
}

// UpdateBug 更新Bug
func (h *BugHandler) UpdateBug(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能更新自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限更新该Bug")
		return
	}

	// 保存旧对象用于比较（深拷贝指针字段，避免修改bug时影响oldBug）
	oldBug := bug
	if bug.RequirementID != nil {
		reqID := *bug.RequirementID
		oldBug.RequirementID = &reqID
	} else {
		oldBug.RequirementID = nil
	}
	if bug.ModuleID != nil {
		modID := *bug.ModuleID
		oldBug.ModuleID = &modID
	} else {
		oldBug.ModuleID = nil
	}

	var req struct {
		Title          *string  `json:"title"`
		Description    *string  `json:"description"`
		Status         *string  `json:"status"`
		Priority       *string  `json:"priority"`
		Severity       *string  `json:"severity"`
		ProjectID      *uint    `json:"project_id"`
		RequirementID  *uint    `json:"requirement_id"`
		ModuleID       *uint    `json:"module_id"`
		AssigneeIDs    *[]uint  `json:"assignee_ids"`
		EstimatedHours *float64 `json:"estimated_hours"`
		ActualHours    *float64 `json:"actual_hours"`   // 实际工时，会自动创建资源分配
		WorkDate       *string  `json:"work_date"`      // 工作日期（YYYY-MM-DD），用于资源分配
		VersionIDs     *[]uint  `json:"version_ids"`    // 所属版本ID列表
		AttachmentIDs  *[]uint  `json:"attachment_ids"` // 附件ID列表
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 更新字段
	if req.Title != nil {
		bug.Title = *req.Title
	}
	// 注意：即使 description 是空字符串，也要更新（使用指针判断，空字符串指针不为 nil）
	if req.Description != nil {
		bug.Description = *req.Description
	}
	if req.Status != nil {
		// 验证状态
		validStatuses := map[string]bool{
			"active":   true,
			"resolved": true,
			"closed":   true,
		}
		if !validStatuses[*req.Status] {
			utils.Error(c, 400, "状态值无效，有效值：active, resolved, closed")
			return
		}

		// 验证状态流转是否符合禅道规则
		// 禅道规则：
		// 1. active -> resolved (只有active状态可以解决)
		// 2. resolved -> closed (只有resolved状态可以关闭)
		// 3. resolved/closed -> active (只有非active状态可以激活)
		currentStatus := bug.Status
		newStatus := *req.Status

		if currentStatus != newStatus {
			if currentStatus == "active" && newStatus == "resolved" {
				// active -> resolved: 允许
			} else if currentStatus == "resolved" && newStatus == "closed" {
				// resolved -> closed: 允许
			} else if (currentStatus == "resolved" || currentStatus == "closed") && newStatus == "active" {
				// resolved/closed -> active: 允许（激活）
			} else {
				// 其他状态转换不允许
				utils.Error(c, 400, fmt.Sprintf("状态转换无效：不能从 %s 转换到 %s。允许的转换：active->resolved, resolved->closed, resolved/closed->active", currentStatus, newStatus))
				return
			}
		}

		bug.Status = *req.Status
	}
	if req.Priority != nil {
		// 验证优先级
		validPriorities := map[string]bool{
			"low":    true,
			"medium": true,
			"high":   true,
			"urgent": true,
		}
		if !validPriorities[*req.Priority] {
			utils.Error(c, 400, "优先级值无效")
			return
		}
		bug.Priority = *req.Priority
	}
	if req.Severity != nil {
		// 验证严重程度
		validSeverities := map[string]bool{
			"low":      true,
			"medium":   true,
			"high":     true,
			"critical": true,
		}
		if !validSeverities[*req.Severity] {
			utils.Error(c, 400, "严重程度值无效")
			return
		}
		bug.Severity = *req.Severity
	}
	if req.ProjectID != nil {
		// 验证项目是否存在
		var project model.Project
		if err := h.db.First(&project, *req.ProjectID).Error; err != nil {
			utils.Error(c, 400, "项目不存在")
			return
		}
		bug.ProjectID = *req.ProjectID
	}
	if req.RequirementID != nil {
		// 验证需求是否存在
		if *req.RequirementID != 0 {
			var requirement model.Requirement
			if err := h.db.First(&requirement, *req.RequirementID).Error; err != nil {
				utils.Error(c, 400, "需求不存在")
				return
			}
			bug.RequirementID = req.RequirementID
		} else {
			bug.RequirementID = nil
		}
	}
	if req.ModuleID != nil {
		// 验证功能模块是否存在
		if *req.ModuleID != 0 {
			var module model.Module
			if err := h.db.First(&module, *req.ModuleID).Error; err != nil {
				utils.Error(c, 400, "功能模块不存在")
				return
			}
			bug.ModuleID = req.ModuleID
		} else {
			bug.ModuleID = nil
		}
	}
	if req.EstimatedHours != nil {
		if *req.EstimatedHours < 0 {
			utils.Error(c, 400, "预估工时不能为负数")
			return
		}
		bug.EstimatedHours = req.EstimatedHours
	}

	// 如果更新了实际工时，自动创建或更新资源分配
	if req.ActualHours != nil {
		if *req.ActualHours < 0 {
			utils.Error(c, 400, "实际工时不能为负数")
			return
		}
		// 先加载分配人信息
		h.db.Preload("Assignees").First(&bug, bug.ID)

		// 注意：重新查询后，需要恢复已更新的字段（如 description）
		if req.Description != nil {
			bug.Description = *req.Description
		}
		if req.Title != nil {
			bug.Title = *req.Title
		}
		// 恢复其他可能已更新的字段
		if req.Status != nil {
			bug.Status = *req.Status
		}
		if req.Priority != nil {
			bug.Priority = *req.Priority
		}
		if req.Severity != nil {
			bug.Severity = *req.Severity
		}
		if req.ProjectID != nil {
			bug.ProjectID = *req.ProjectID
		}
		if req.RequirementID != nil {
			if *req.RequirementID == 0 {
				bug.RequirementID = nil
			} else {
				bug.RequirementID = req.RequirementID
			}
		}
		if req.ModuleID != nil {
			if *req.ModuleID == 0 {
				bug.ModuleID = nil
			} else {
				bug.ModuleID = req.ModuleID
			}
		}
		if req.EstimatedHours != nil {
			bug.EstimatedHours = req.EstimatedHours
		}

		// Bug可能有多个分配人，需要为每个分配人创建资源分配
		// 这里先处理第一个分配人，或者需要前端指定分配人
		// 暂时使用第一个分配人，如果没有分配人则直接设置actual_hours
		if len(bug.Assignees) > 0 {
			// 确定工作日期
			var workDate time.Time
			if req.WorkDate != nil && *req.WorkDate != "" {
				if t, err := time.Parse("2006-01-02", *req.WorkDate); err == nil {
					workDate = t
				} else {
					utils.Error(c, 400, "工作日期格式错误，应为 YYYY-MM-DD")
					return
				}
			} else {
				workDate = time.Now()
			}
			workDate = time.Date(workDate.Year(), workDate.Month(), workDate.Day(), 0, 0, 0, 0, workDate.Location())

			// 为第一个分配人同步到资源分配
			if err := h.syncBugActualHours(&bug, *req.ActualHours, workDate, bug.Assignees[0].ID); err != nil {
				utils.Error(c, utils.CodeError, "同步资源分配失败: "+err.Error())
				return
			}
			// 从资源分配中汇总实际工时（确保actual_hours正确）
			h.calculateAndUpdateActualHours(&bug)
		} else {
			// 如果没有分配人，直接设置actual_hours，但不创建资源分配
			bug.ActualHours = req.ActualHours
		}
	}

	if err := h.db.Save(&bug).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 更新分配人
	if req.AssigneeIDs != nil {
		var assignees []model.User
		if len(*req.AssigneeIDs) > 0 {
			if err := h.db.Where("id IN ?", *req.AssigneeIDs).Find(&assignees).Error; err != nil || len(assignees) != len(*req.AssigneeIDs) {
				utils.Error(c, 400, "分配人不存在")
				return
			}
		}
		if err := h.db.Model(&bug).Association("Assignees").Replace(assignees); err != nil {
			utils.Error(c, utils.CodeError, "更新分配失败")
			return
		}
		// 注意：禅道中Bug只有active/resolved/closed三种状态
		// 分配Bug不会自动改变状态，状态需要手动更新
	}

	// 更新版本关联
	if req.VersionIDs != nil {
		// 验证版本是否存在且属于同一项目
		projectID := bug.ProjectID
		if req.ProjectID != nil {
			projectID = *req.ProjectID
		}
		if len(*req.VersionIDs) == 0 {
			utils.Error(c, 400, "必须至少选择一个所属版本")
			return
		}
		var versions []model.Version
		if err := h.db.Where("id IN ? AND project_id = ?", *req.VersionIDs, projectID).Find(&versions).Error; err != nil {
			utils.Error(c, 400, "版本查询失败")
			return
		}
		if len(versions) != len(*req.VersionIDs) {
			utils.Error(c, 400, "版本不存在或不属于当前项目")
			return
		}
		if err := h.db.Model(&bug).Association("Versions").Replace(versions); err != nil {
			utils.Error(c, utils.CodeError, "更新版本关联失败")
			return
		}
	}

	// 更新附件关联
	if req.AttachmentIDs != nil {
		projectID := bug.ProjectID
		if req.ProjectID != nil {
			projectID = *req.ProjectID
		}
		var attachments []model.Attachment
		if len(*req.AttachmentIDs) > 0 {
			// 验证附件是否存在且属于同一项目
			// 注意：附件可能已经关联到项目，也可能还没有，所以先查询附件是否存在
			// 注意：使用 Unscoped() 查询所有附件（包括软删除的），但只关联未删除的附件
			if err := h.db.Where("id IN ?", *req.AttachmentIDs).Find(&attachments).Error; err != nil {
				utils.Error(c, 400, "附件查询失败: "+err.Error())
				return
			}
			if len(attachments) != len(*req.AttachmentIDs) {
				// 检查是否有附件被软删除
				var deletedAttachments []model.Attachment
				h.db.Unscoped().Where("id IN ? AND deleted_at IS NOT NULL", *req.AttachmentIDs).Find(&deletedAttachments)
				if len(deletedAttachments) > 0 {
					utils.Error(c, 400, fmt.Sprintf("部分附件已被删除：期望 %d 个，实际找到 %d 个，已删除 %d 个", len(*req.AttachmentIDs), len(attachments), len(deletedAttachments)))
					return
				}
				utils.Error(c, 400, fmt.Sprintf("附件不存在：期望 %d 个，实际找到 %d 个", len(*req.AttachmentIDs), len(attachments)))
				return
			}
			// 验证附件是否属于同一项目（通过检查附件是否关联到项目）
			for _, attachment := range attachments {
				var count int64
				if err := h.db.Table("project_attachments").
					Where("attachment_id = ? AND project_id = ?", attachment.ID, projectID).
					Count(&count).Error; err != nil {
					utils.Error(c, 400, "验证附件项目关联失败: "+err.Error())
					return
				}
				if count == 0 {
					utils.Error(c, 400, fmt.Sprintf("附件 %d 不属于项目 %d", attachment.ID, projectID))
					return
				}
			}
		}
		// 使用Replace方法同步附件关联（空数组表示移除所有附件）
		if err := h.db.Model(&bug).Association("Attachments").Replace(attachments); err != nil {
			utils.Error(c, utils.CodeError, "更新附件关联失败: "+err.Error())
			return
		}
	}

	// 重新加载关联数据（包括附件）
	// 注意：必须在 Replace 之后重新查询，才能获取到最新的附件关联
	// 使用 Session 确保使用新的查询上下文，避免缓存问题
	// 注意：Preload 会自动过滤软删除的记录（DeletedAt IS NULL）
	if err := h.db.Session(&gorm.Session{}).Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("ResolvedVersion").Preload("Versions").Preload("Attachments").Preload("Attachments.Creator").First(&bug, bug.ID).Error; err != nil {
		utils.Error(c, utils.CodeError, "重新加载Bug数据失败: "+err.Error())
		return
	}

	// 记录编辑操作和字段变更
	userID, exists := c.Get("user_id")
	if exists {
		dbValue, _ := c.Get("db")
		if db, ok := dbValue.(*gorm.DB); ok {
			// 比较新旧对象并记录变更
			utils.CompareAndRecord(db, oldBug, bug, "bug", bug.ID, userID.(uint), "edited")
		}
	}

	utils.Success(c, bug)
}

// DeleteBug 删除Bug
func (h *BugHandler) DeleteBug(c *gin.Context) {
	id := c.Param("id")

	// 验证Bug是否存在
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能删除自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限删除该Bug")
		return
	}

	if err := h.db.Delete(&model.Bug{}, id).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除失败")
		return
	}

	utils.Success(c, gin.H{"message": "删除成功"})
}

// UpdateBugStatus 更新Bug状态
func (h *BugHandler) UpdateBugStatus(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能更新自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限更新该Bug")
		return
	}

	// 保存旧对象用于比较
	oldBug := bug

	var req struct {
		Status            string   `json:"status" binding:"required"`
		Solution          *string  `json:"solution"`            // 解决方案
		SolutionNote      *string  `json:"solution_note"`       // 解决方案备注
		EstimatedHours    *float64 `json:"estimated_hours"`     // 预估工时
		ActualHours       *float64 `json:"actual_hours"`        // 实际工时
		WorkDate          *string  `json:"work_date"`           // 工作日期（YYYY-MM-DD），用于资源分配
		ResolvedVersionID *uint    `json:"resolved_version_id"` // 解决版本ID
		VersionNumber     *string  `json:"version_number"`      // 版本号（如果创建新版本）
		CreateVersion     *bool    `json:"create_version"`      // 是否创建新版本
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 验证状态
	validStatuses := map[string]bool{
		"active":   true,
		"resolved": true,
		"closed":   true,
	}
	if !validStatuses[req.Status] {
		utils.Error(c, 400, "状态值无效，有效值：active, resolved, closed")
		return
	}

	// 验证状态流转是否符合禅道规则
	// 禅道规则：
	// 1. active -> resolved (只有active状态可以解决)
	// 2. resolved -> closed (只有resolved状态可以关闭)
	// 3. resolved/closed -> active (只有非active状态可以激活)
	currentStatus := bug.Status
	newStatus := req.Status

	if currentStatus == newStatus {
		// 状态未改变，允许
	} else if currentStatus == "active" && newStatus == "resolved" {
		// active -> resolved: 允许
	} else if currentStatus == "resolved" && newStatus == "closed" {
		// resolved -> closed: 允许
	} else if (currentStatus == "resolved" || currentStatus == "closed") && newStatus == "active" {
		// resolved/closed -> active: 允许（激活）
	} else {
		// 其他状态转换不允许
		utils.Error(c, 400, fmt.Sprintf("状态转换无效：不能从 %s 转换到 %s。允许的转换：active->resolved, resolved->closed, resolved/closed->active", currentStatus, newStatus))
		return
	}

	// 验证解决方案（如果提供了）
	if req.Solution != nil {
		validSolutions := map[string]bool{
			"设计如此":   true,
			"重复Bug":  true,
			"外部原因":   true,
			"已解决":    true,
			"无法重现":   true,
			"延期处理":   true,
			"不予解决":   true,
			"转为研发需求": true,
		}
		if !validSolutions[*req.Solution] {
			utils.Error(c, 400, "解决方案值无效")
			return
		}
		bug.Solution = *req.Solution
	}

	// 更新解决方案备注
	if req.SolutionNote != nil {
		bug.SolutionNote = *req.SolutionNote
	}

	// 禅道逻辑：当有解决方案时，自动确认Bug
	// 如果状态变为resolved且有解决方案，自动设置为已确认
	if req.Status == "resolved" && req.Solution != nil && *req.Solution != "" {
		bug.Confirmed = true
	}

	// 禅道逻辑：当状态变为resolved时，自动指派给创建者
	var autoAssigned bool
	var autoAssignedUserIDs []uint
	var oldAssigneeIDsForHistory []uint
	if req.Status == "resolved" && currentStatus != "resolved" {
		// 加载当前分配人信息
		h.db.Preload("Assignees").First(&bug, bug.ID)

		// 获取旧的分配人ID列表（用于记录历史）
		for _, assignee := range bug.Assignees {
			oldAssigneeIDsForHistory = append(oldAssigneeIDsForHistory, assignee.ID)
		}

		// 直接指派给创建者
		assigneeIDs := []uint{bug.CreatorID}

		// 验证创建者是否存在
		var assignees []model.User
		if err := h.db.Where("id IN ?", assigneeIDs).Find(&assignees).Error; err == nil && len(assignees) > 0 {
			// 自动指派
			if err := h.db.Model(&bug).Association("Assignees").Replace(assignees); err == nil {
				// 记录自动指派信息，用于后续记录历史
				autoAssigned = true
				autoAssignedUserIDs = assigneeIDs
				// 重新加载分配人信息
				h.db.Preload("Assignees").First(&bug, bug.ID)
			}
		}
	}

	// 处理版本号
	var resolvedVersionID *uint
	if req.CreateVersion != nil && *req.CreateVersion && req.VersionNumber != nil && *req.VersionNumber != "" {
		// 创建新版本
		version := model.Version{
			VersionNumber: *req.VersionNumber,
			ReleaseNotes:  "Bug修复版本",
			Status:        "wait",
			ProjectID:     bug.ProjectID,
		}
		if err := h.db.Create(&version).Error; err != nil {
			utils.Error(c, utils.CodeError, "创建版本失败")
			return
		}
		// 关联当前Bug到新版本
		h.db.Model(&version).Association("Bugs").Append(&bug)
		resolvedVersionID = &version.ID
	} else if req.ResolvedVersionID != nil {
		// 使用已有版本
		// 验证版本是否存在且属于同一项目
		var version model.Version
		if err := h.db.First(&version, *req.ResolvedVersionID).Error; err != nil {
			utils.Error(c, 400, "版本不存在")
			return
		}
		if version.ProjectID != bug.ProjectID {
			utils.Error(c, 400, "版本必须属于同一项目")
			return
		}
		resolvedVersionID = req.ResolvedVersionID
		// 关联当前Bug到版本
		h.db.Model(&version).Association("Bugs").Append(&bug)
	}

	if resolvedVersionID != nil {
		bug.ResolvedVersionID = resolvedVersionID
	}

	// 更新预估工时
	if req.EstimatedHours != nil {
		if *req.EstimatedHours < 0 {
			utils.Error(c, 400, "预估工时不能为负数")
			return
		}
		bug.EstimatedHours = req.EstimatedHours
	}

	// 更新实际工时（如果提供了）
	if req.ActualHours != nil {
		if *req.ActualHours < 0 {
			utils.Error(c, 400, "实际工时不能为负数")
			return
		}
		// 先加载分配人信息
		h.db.Preload("Assignees").First(&bug, bug.ID)

		// 如果有分配人，创建或更新资源分配
		if len(bug.Assignees) > 0 {
			// 确定工作日期
			var workDate time.Time
			if req.WorkDate != nil && *req.WorkDate != "" {
				if t, err := time.Parse("2006-01-02", *req.WorkDate); err == nil {
					workDate = t
				} else {
					utils.Error(c, 400, "工作日期格式错误，应为 YYYY-MM-DD")
					return
				}
			} else {
				workDate = time.Now()
			}
			workDate = time.Date(workDate.Year(), workDate.Month(), workDate.Day(), 0, 0, 0, 0, workDate.Location())

			// 为第一个分配人同步到资源分配
			if err := h.syncBugActualHours(&bug, *req.ActualHours, workDate, bug.Assignees[0].ID); err != nil {
				utils.Error(c, utils.CodeError, "同步资源分配失败: "+err.Error())
				return
			}
			// 从资源分配中汇总实际工时（确保actual_hours正确）
			h.calculateAndUpdateActualHours(&bug)
		} else {
			// 如果没有分配人，直接设置actual_hours，但不创建资源分配
			bug.ActualHours = req.ActualHours
		}
	}

	bug.Status = req.Status
	if err := h.db.Save(&bug).Error; err != nil {
		utils.Error(c, utils.CodeError, "更新失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("ResolvedVersion").First(&bug, bug.ID)

	// 记录解决/关闭操作和字段变更
	userID, exists := c.Get("user_id")
	if exists {
		dbValue, _ := c.Get("db")
		if db, ok := dbValue.(*gorm.DB); ok {
			actionType := "resolved"
			if req.Status == "closed" {
				actionType = "closed"
			}
			// 准备extra信息（包含解决方案等）
			extra := make(map[string]interface{})
			if req.Solution != nil {
				extra["solution"] = *req.Solution
			}
			if req.ResolvedVersionID != nil {
				extra["resolved_version_id"] = *req.ResolvedVersionID
			}
			// 记录操作（包含备注）
			comment := ""
			if req.SolutionNote != nil {
				comment = *req.SolutionNote
			}
			// 使用CompareAndRecord会自动记录操作和字段变更，但我们需要先记录操作以包含extra信息
			// 所以先记录操作，然后记录字段变更
			actionID, _ := utils.RecordAction(db, "bug", bug.ID, actionType, userID.(uint), comment, extra)
			// 记录字段变更
			changes := utils.CompareObjects(oldBug, bug)
			// 如果自动指派了，手动添加assignee_ids的变更记录（因为CompareObjects不会比较关联字段）
			if autoAssigned && len(autoAssignedUserIDs) > 0 {
				// 检查是否已经有assignee_ids的变更记录
				hasAssigneeChange := false
				for _, change := range changes {
					if change.Field == "assignee_ids" {
						hasAssigneeChange = true
						break
					}
				}
				// 如果没有，添加自动指派的变更记录
				if !hasAssigneeChange {
					oldIDsStr := formatUintSlice(oldAssigneeIDsForHistory)
					newIDsStr := formatUintSlice(autoAssignedUserIDs)
					if oldIDsStr != newIDsStr {
						changes = append(changes, utils.HistoryChange{
							Field: "assignee_ids",
							Old:   oldIDsStr,
							New:   newIDsStr,
						})
					}
				}
			}
			if len(changes) > 0 {
				utils.RecordHistory(db, actionID, changes)
			}
		}
	}

	utils.Success(c, bug)
}

// GetBugStatistics 获取Bug统计
func (h *BugHandler) GetBugStatistics(c *gin.Context) {
	var stats struct {
		Total            int64 `json:"total"`
		Active           int64 `json:"active"`   // 激活状态（对应原来的open）
		Resolved         int64 `json:"resolved"` // 已解决
		Closed           int64 `json:"closed"`   // 已关闭
		LowPriority      int64 `json:"low_priority"`
		MediumPriority   int64 `json:"medium_priority"`
		HighPriority     int64 `json:"high_priority"`
		UrgentPriority   int64 `json:"urgent_priority"`
		LowSeverity      int64 `json:"low_severity"`
		MediumSeverity   int64 `json:"medium_severity"`
		HighSeverity     int64 `json:"high_severity"`
		CriticalSeverity int64 `json:"critical_severity"`
	}

	baseQuery := h.db.Model(&model.Bug{})

	// 权限过滤：普通用户只能看到自己创建或参与的Bug
	baseQuery = utils.FilterBugsByUser(h.db, c, baseQuery)

	// 应用筛选条件（与列表查询保持一致）
	if keyword := c.Query("keyword"); keyword != "" {
		baseQuery = baseQuery.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if projectID := c.Query("project_id"); projectID != "" {
		baseQuery = baseQuery.Where("project_id = ?", projectID)
	}
	if requirementID := c.Query("requirement_id"); requirementID != "" {
		baseQuery = baseQuery.Where("requirement_id = ?", requirementID)
	}
	if creatorID := c.Query("creator_id"); creatorID != "" {
		baseQuery = baseQuery.Where("creator_id = ?", creatorID)
	}

	// 统计总数
	baseQuery.Session(&gorm.Session{}).Count(&stats.Total)

	// 按状态统计（禅道状态：active, resolved, closed）
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "active").Count(&stats.Active)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "resolved").Count(&stats.Resolved)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "closed").Count(&stats.Closed)

	// 按优先级统计
	baseQuery.Session(&gorm.Session{}).Where("priority = ?", "low").Count(&stats.LowPriority)
	baseQuery.Session(&gorm.Session{}).Where("priority = ?", "medium").Count(&stats.MediumPriority)
	baseQuery.Session(&gorm.Session{}).Where("priority = ?", "high").Count(&stats.HighPriority)
	baseQuery.Session(&gorm.Session{}).Where("priority = ?", "urgent").Count(&stats.UrgentPriority)

	// 按严重程度统计
	baseQuery.Session(&gorm.Session{}).Where("severity = ?", "low").Count(&stats.LowSeverity)
	baseQuery.Session(&gorm.Session{}).Where("severity = ?", "medium").Count(&stats.MediumSeverity)
	baseQuery.Session(&gorm.Session{}).Where("severity = ?", "high").Count(&stats.HighSeverity)
	baseQuery.Session(&gorm.Session{}).Where("severity = ?", "critical").Count(&stats.CriticalSeverity)

	utils.Success(c, stats)
}

// AssignBug 分配Bug给用户
func (h *BugHandler) AssignBug(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能分配自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限分配该Bug")
		return
	}

	// 获取旧的分配人ID列表
	var oldAssigneeIDs []uint
	h.db.Model(&model.BugAssignee{}).Where("bug_id = ?", bug.ID).Pluck("user_id", &oldAssigneeIDs)

	var req struct {
		AssigneeIDs []uint  `json:"assignee_ids" binding:"required"`
		Status      *string `json:"status"`
		Comment     *string `json:"comment"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 验证用户是否存在
	var users []model.User
	if err := h.db.Where("id IN ?", req.AssigneeIDs).Find(&users).Error; err != nil || len(users) != len(req.AssigneeIDs) {
		utils.Error(c, 400, "用户不存在")
		return
	}

	// 分配Bug
	if err := h.db.Model(&bug).Association("Assignees").Replace(users); err != nil {
		utils.Error(c, utils.CodeError, "分配失败")
		return
	}

	// 如果提供了状态，更新Bug状态
	if req.Status != nil {
		validStatuses := map[string]bool{"active": true, "resolved": true, "closed": true}
		if !validStatuses[*req.Status] {
			utils.Error(c, 400, "无效的状态值")
			return
		}
		bug.Status = *req.Status
		if err := h.db.Model(&bug).Update("status", *req.Status).Error; err != nil {
			utils.Error(c, utils.CodeError, "更新状态失败")
			return
		}
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("ResolvedVersion").First(&bug, bug.ID)

	// 记录分配操作
	userID, exists := c.Get("user_id")
	if exists {
		dbValue, _ := c.Get("db")
		if db, ok := dbValue.(*gorm.DB); ok {
			// 记录分配操作
			actionID, _ := utils.RecordAction(db, "bug", bug.ID, "assigned", userID.(uint), "", nil)
			// 记录指派字段变更
			oldIDsStr := formatUintSlice(oldAssigneeIDs)
			newIDsStr := formatUintSlice(req.AssigneeIDs)
			if oldIDsStr != newIDsStr {
				changes := []utils.HistoryChange{
					{Field: "assignee_ids", Old: oldIDsStr, New: newIDsStr},
				}
				utils.RecordHistory(db, actionID, changes)
			}

			// 如果提供了备注，记录备注操作
			if req.Comment != nil && *req.Comment != "" {
				_, err := utils.RecordAction(db, "bug", bug.ID, "commented", userID.(uint), *req.Comment, nil)
				if err != nil {
					utils.Error(c, utils.CodeError, "添加备注失败")
					return
				}
			}
		}
	}

	utils.Success(c, bug)
}

// formatUintSlice 格式化uint切片为字符串
func formatUintSlice(ids []uint) string {
	if len(ids) == 0 {
		return ""
	}
	str := fmt.Sprintf("%d", ids[0])
	for i := 1; i < len(ids); i++ {
		str += fmt.Sprintf(",%d", ids[i])
	}
	return str
}

// parseUintSlice 解析逗号分隔的用户ID字符串为uint切片
func parseUintSlice(idsStr string) []uint {
	if idsStr == "" {
		return nil
	}
	parts := strings.Split(idsStr, ",")
	var ids []uint
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 32)
		if err == nil && id > 0 {
			ids = append(ids, uint(id))
		}
	}
	return ids
}

// getLastAssignedUserIDs 从历史记录中查找最后一个被分配的用户ID列表（已废弃，使用getLastAssignedTesterIDs）
func (h *BugHandler) getLastAssignedUserIDs(bugID uint) []uint {
	// 直接查询历史记录，查找assignee_ids字段的变更，按时间倒序
	// 使用JOIN查询，确保按操作时间排序
	var histories []model.History
	if err := h.db.Table("histories").
		Select("histories.*").
		Joins("JOIN actions ON actions.id = histories.action_id").
		Where("actions.object_type = ? AND actions.object_id = ? AND histories.field = ? AND histories.new != '' AND histories.new IS NOT NULL", "bug", bugID, "assignee_ids").
		Order("actions.date DESC, actions.id DESC").
		Limit(1).
		Find(&histories).Error; err != nil {
		return nil
	}

	// 如果找到了历史记录
	if len(histories) > 0 {
		history := histories[0]
		userIDs := parseUintSlice(history.New)
		if len(userIDs) > 0 {
			// 验证这些用户是否还存在
			var validUsers []model.User
			if err := h.db.Where("id IN ?", userIDs).Find(&validUsers).Error; err == nil && len(validUsers) > 0 {
				// 返回有效的用户ID列表
				validIDs := make([]uint, 0, len(validUsers))
				for _, user := range validUsers {
					validIDs = append(validIDs, user.ID)
				}
				return validIDs
			}
		}
	}

	return nil
}

// getLastAssignedTesterIDs 从历史记录中查找最后一个被分配的测试工程师ID列表
// 排除当前用户（解决Bug的工程师）
func (h *BugHandler) getLastAssignedTesterIDs(bugID uint, excludeUserID uint) []uint {
	// 直接查询历史记录，查找assignee_ids字段的变更，按时间倒序
	// 使用JOIN查询，确保按操作时间排序
	var histories []model.History
	if err := h.db.Table("histories").
		Select("histories.*").
		Joins("JOIN actions ON actions.id = histories.action_id").
		Where("actions.object_type = ? AND actions.object_id = ? AND histories.field = ? AND histories.new != '' AND histories.new IS NOT NULL", "bug", bugID, "assignee_ids").
		Order("actions.date DESC, actions.id DESC").
		Find(&histories).Error; err != nil {
		return nil
	}

	// 遍历历史记录，查找测试工程师
	for _, history := range histories {
		userIDs := parseUintSlice(history.New)
		if len(userIDs) > 0 {
			// 排除当前用户（解决Bug的工程师）
			filteredUserIDs := make([]uint, 0, len(userIDs))
			for _, id := range userIDs {
				if id != excludeUserID {
					filteredUserIDs = append(filteredUserIDs, id)
				}
			}

			if len(filteredUserIDs) == 0 {
				continue
			}

			// 查询这些用户，并检查是否有测试工程师角色
			var users []model.User
			if err := h.db.Preload("Roles").Where("id IN ?", filteredUserIDs).Find(&users).Error; err == nil && len(users) > 0 {
				// 查找测试工程师（角色代码为 "tester"）
				testerIDs := make([]uint, 0)
				for _, user := range users {
					for _, role := range user.Roles {
						if role.Code == "tester" {
							testerIDs = append(testerIDs, user.ID)
							break
						}
					}
				}

				// 如果找到了测试工程师，返回他们的ID列表
				if len(testerIDs) > 0 {
					return testerIDs
				}
			}
		}
	}

	return nil
}

// ConfirmBug 确认Bug
func (h *BugHandler) ConfirmBug(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能确认自己创建或参与的Bug
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限确认该Bug")
		return
	}

	// 验证：只有状态为active且未确认的bug才能被确认
	if bug.Status != "active" {
		utils.Error(c, 400, "只有激活状态的Bug才能被确认")
		return
	}

	if bug.Confirmed {
		utils.Error(c, 400, "该Bug已经确认过了")
		return
	}

	// 确认Bug
	bug.Confirmed = true
	if err := h.db.Save(&bug).Error; err != nil {
		utils.Error(c, utils.CodeError, "确认失败")
		return
	}

	// 重新加载关联数据
	h.db.Preload("Project").Preload("Creator").Preload("Assignees").Preload("Requirement").Preload("Module").Preload("ResolvedVersion").First(&bug, bug.ID)

	// 记录确认操作
	userID, exists := c.Get("user_id")
	if exists {
		dbValue, _ := c.Get("db")
		if db, ok := dbValue.(*gorm.DB); ok {
			utils.RecordAction(db, "bug", bug.ID, "confirmed", userID.(uint), "", nil)
		}
	}

	utils.Success(c, bug)
}

// syncBugActualHours 同步Bug实际工时到资源分配
// 使用事务和 FirstOrCreate 防止并发死锁
func (h *BugHandler) syncBugActualHours(bug *model.Bug, actualHours float64, workDate time.Time, assigneeID uint) error {
	// 使用事务包裹所有操作，防止死锁
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 使用 FirstOrCreate 查找或创建资源，避免并发创建冲突
	var resource model.Resource
	err := tx.Where("user_id = ? AND project_id = ?", assigneeID, bug.ProjectID).
		FirstOrCreate(&resource, model.Resource{
			UserID:    assigneeID,
			ProjectID: bug.ProjectID,
		}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("查找或创建资源失败: %w", err)
	}

	// 使用 FirstOrCreate 查找或创建资源分配，避免并发创建冲突
	var allocation model.ResourceAllocation
	err = tx.Where("resource_id = ? AND bug_id = ? AND date = ?", resource.ID, bug.ID, workDate).
		FirstOrCreate(&allocation, model.ResourceAllocation{
			ResourceID:  resource.ID,
			BugID:       &bug.ID,
			ProjectID:   &bug.ProjectID,
			Date:        workDate,
			Hours:       actualHours,
			Description: fmt.Sprintf("Bug: %s", bug.Title),
		}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("查找或创建资源分配失败: %w", err)
	}

	// 无论记录是新创建还是已存在，都更新工时和描述（确保数据同步）
	allocation.Hours = actualHours
	allocation.Description = fmt.Sprintf("Bug: %s", bug.Title)
	if err := tx.Save(&allocation).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新资源分配失败: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// calculateAndUpdateActualHours 计算并更新Bug的实际工时（从资源分配中汇总）
// 使用事务包裹查询和更新操作，防止并发死锁
func (h *BugHandler) calculateAndUpdateActualHours(bug *model.Bug) {
	// 使用事务包裹查询和更新操作
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var totalHours float64
	if err := tx.Model(&model.ResourceAllocation{}).
		Where("bug_id = ?", bug.ID).
		Select("COALESCE(SUM(hours), 0)").
		Scan(&totalHours).Error; err != nil {
		tx.Rollback()
		return // 查询失败时静默返回，避免影响主流程
	}

	bug.ActualHours = &totalHours
	if err := tx.Model(bug).Update("actual_hours", totalHours).Error; err != nil {
		tx.Rollback()
		return // 更新失败时静默返回，避免影响主流程
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		// 提交失败时静默返回，避免影响主流程
		return
	}
}

// GetBugHistory 获取Bug历史记录列表（参考禅道的 getList() 方法）
func (h *BugHandler) GetBugHistory(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能查看自己创建或参与的Bug的历史记录
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限查看该Bug的历史记录")
		return
	}

	// 查询操作记录
	var actions []model.Action
	if err := h.db.Where("object_type = ? AND object_id = ?", "bug", id).
		Preload("Actor").
		Preload("Histories").
		Order("date DESC").
		Find(&actions).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询历史记录失败")
		return
	}

	// 处理历史记录，转换字段值显示
	for i := range actions {
		for j := range actions[i].Histories {
			processedHistory := utils.ProcessHistory(h.db, &actions[i].Histories[j])
			actions[i].Histories[j] = *processedHistory
		}
	}

	utils.Success(c, gin.H{
		"list": actions,
	})
}

// AddBugHistoryNote 添加备注（参考禅道的 commented 操作）
func (h *BugHandler) AddBugHistoryNote(c *gin.Context) {
	id := c.Param("id")
	var bug model.Bug
	if err := h.db.First(&bug, id).Error; err != nil {
		utils.Error(c, 404, "Bug不存在")
		return
	}

	// 权限检查：普通用户只能为自己创建或参与的Bug添加备注
	if !utils.CheckBugAccess(h.db, c, bug.ID) {
		utils.Error(c, 403, "没有权限为该Bug添加备注")
		return
	}

	var req struct {
		Comment string `json:"comment" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误")
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未登录")
		return
	}

	// 记录备注操作
	dbValue, _ := c.Get("db")
	if db, ok := dbValue.(*gorm.DB); ok {
		_, err := utils.RecordAction(db, "bug", bug.ID, "commented", userID.(uint), req.Comment, nil)
		if err != nil {
			utils.Error(c, utils.CodeError, "添加备注失败")
			return
		}
	}

	utils.Success(c, gin.H{"message": "添加备注成功"})
}

// GetBugColumnSettings 获取Bug列表列设置
func (h *BugHandler) GetBugColumnSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	uid := userID.(uint)

	// 使用原生 SQL 查询，直接读取数据库中的值，避免 GORM 的默认值干扰
	type ColumnSettingRow struct {
		ColumnKey string
		Visible   bool
		Order     int
		Width     *int
	}

	var rows []ColumnSettingRow
	// 使用子查询去重，只取每个 column_key 的最新记录（按 id 降序），并过滤软删除的记录
	if err := h.db.Raw(`
		SELECT column_key, visible, `+"`order`"+`, width 
		FROM user_table_column_settings 
		WHERE id IN (
			SELECT MAX(id) 
			FROM user_table_column_settings 
			WHERE user_id = ? AND page = ? AND deleted_at IS NULL 
			GROUP BY column_key
		)
		ORDER BY `+"`order`"+` ASC
	`, uid, "bug").Scan(&rows).Error; err != nil {
		utils.Error(c, utils.CodeError, "查询列设置失败")
		return
	}

	// 转换为前端需要的格式
	result := make([]gin.H, len(rows))
	for i, row := range rows {
		item := gin.H{
			"key":     row.ColumnKey,
			"visible": row.Visible, // 直接使用从数据库读取的值
			"order":   row.Order,
		}
		if row.Width != nil {
			item["width"] = *row.Width
		}
		result[i] = item
	}

	utils.Success(c, result)
}

// SaveBugColumnSettings 保存Bug列表列设置
func (h *BugHandler) SaveBugColumnSettings(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Error(c, 401, "未授权")
		return
	}

	uid := userID.(uint)

	var req []struct {
		Key     string `json:"key" binding:"required"`
		Visible bool   `json:"visible"`
		Order   int    `json:"order"`
		Width   *int   `json:"width,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c, 400, "参数错误: "+err.Error())
		return
	}

	// 开启事务
	tx := h.db.Begin()
	if tx.Error != nil {
		utils.Error(c, utils.CodeError, "开启事务失败")
		return
	}

	// 使用 defer 确保事务被正确处理
	committed := false
	defer func() {
		if !committed {
			if r := recover(); r != nil {
				tx.Rollback()
				panic(r) // 重新抛出 panic
			} else {
				tx.Rollback()
			}
		}
	}()

	// 删除该用户该页面的所有现有设置（硬删除，包括软删除的记录）
	if err := tx.Unscoped().Where("user_id = ? AND page = ?", uid, "bug").Delete(&model.UserTableColumnSetting{}).Error; err != nil {
		utils.Error(c, utils.CodeError, "删除旧设置失败: "+err.Error())
		return
	}

	// 创建新设置
	now := time.Now()
	for _, item := range req {
		// 使用原生 SQL 插入，确保 visible 字段（包括 false）被正确保存
		// 这样可以避免 GORM 的默认值干扰
		var widthValue interface{}
		if item.Width != nil {
			widthValue = *item.Width
		} else {
			widthValue = nil
		}

		// 使用参数化查询，兼容 SQLite 和 MySQL
		if err := tx.Exec(`
			INSERT INTO user_table_column_settings (user_id, page, column_key, visible, `+"`order`"+`, width, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, uid, "bug", item.Key, item.Visible, item.Order, widthValue, now, now).Error; err != nil {
			utils.Error(c, utils.CodeError, "保存列设置失败: "+err.Error())
			return
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		utils.Error(c, utils.CodeError, "提交事务失败: "+err.Error())
		return
	}
	committed = true

	utils.Success(c, gin.H{"message": "列设置已保存"})
}
