<template>
  <div class="bug-management">
    <a-layout>
      <AppHeader />
      <a-layout-content class="content">
        <div class="content-inner">
          <a-tabs v-model:activeKey="activeTab">
            <!-- 统计标签页 -->
            <a-tab-pane key="statistics" tab="统计">
              <!-- 统计概览 -->
              <a-row :gutter="16" style="margin-bottom: 16px">
            <a-col :span="6">
              <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick()">
                <a-statistic
                  title="总Bug数"
                  :value="statistics?.total || 0"
                  :value-style="{ color: '#ff4d4f' }"
                />
              </a-card>
            </a-col>
            <a-col :span="6">
              <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick('active')">
                <a-statistic
                  title="激活"
                  :value="statistics?.active || 0"
                  :value-style="{ color: '#faad14' }"
                />
              </a-card>
            </a-col>
            <a-col :span="6">
              <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick('resolved')">
                <a-statistic
                  title="已解决"
                  :value="statistics?.resolved || 0"
                  :value-style="{ color: '#1890ff' }"
                />
              </a-card>
            </a-col>
            <a-col :span="6">
              <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick('closed')">
                <a-statistic
                  title="已关闭"
                  :value="statistics?.closed || 0"
                  :value-style="{ color: '#52c41a' }"
                />
              </a-card>
            </a-col>
          </a-row>

          <!-- 优先级和严重程度统计 -->
          <a-row :gutter="16" style="margin-bottom: 16px">
            <a-col :span="12">
              <a-card title="优先级统计" :bordered="false">
                <a-row :gutter="16">
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, 'low')">
                      <a-statistic
                        title="低"
                        :value="statistics?.low_priority || 0"
                        :value-style="{ color: '#8c8c8c' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, 'medium')">
                      <a-statistic
                        title="中"
                        :value="statistics?.medium_priority || 0"
                        :value-style="{ color: '#1890ff' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, 'high')">
                      <a-statistic
                        title="高"
                        :value="statistics?.high_priority || 0"
                        :value-style="{ color: '#faad14' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, 'urgent')">
                      <a-statistic
                        title="紧急"
                        :value="statistics?.urgent_priority || 0"
                        :value-style="{ color: '#ff4d4f' }"
                      />
                    </a-card>
                  </a-col>
                </a-row>
              </a-card>
            </a-col>
            <a-col :span="12">
              <a-card title="严重程度统计" :bordered="false">
                <a-row :gutter="16">
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, undefined, 'low')">
                      <a-statistic
                        title="低"
                        :value="statistics?.low_severity || 0"
                        :value-style="{ color: '#8c8c8c' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, undefined, 'medium')">
                      <a-statistic
                        title="中"
                        :value="statistics?.medium_severity || 0"
                        :value-style="{ color: '#1890ff' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, undefined, 'high')">
                      <a-statistic
                        title="高"
                        :value="statistics?.high_severity || 0"
                        :value-style="{ color: '#faad14' }"
                      />
                    </a-card>
                  </a-col>
                  <a-col :span="6">
                    <a-card :bordered="false" class="statistic-card-clickable" @click="handleStatisticClick(undefined, undefined, 'critical')">
                      <a-statistic
                        title="严重"
                        :value="statistics?.critical_severity || 0"
                        :value-style="{ color: '#ff4d4f' }"
                      />
                    </a-card>
                  </a-col>
                </a-row>
              </a-card>
            </a-col>
          </a-row>
            </a-tab-pane>

            <!-- 列表标签页 -->
            <a-tab-pane key="list" tab="列表">
              <a-card :bordered="false" style="margin-bottom: 0">
                <template #title>
                  <a-space style="width: 100%; justify-content: space-between">
                    <a-space>
                      <span>搜索条件</span>
                      <a-button type="text" size="small" @click="toggleSearchForm">
                        <template #icon>
                          <UpOutlined v-if="searchFormVisible" />
                          <DownOutlined v-else />
                        </template>
                        {{ searchFormVisible ? '收起' : '展开' }}
                      </a-button>
                    </a-space>
                    <a-space>
                      <a-button 
                        type="text" 
                        @click="showColumnSettingsModal"
                      >
                        <template #icon><SettingOutlined /></template>
                        列设置
                      </a-button>
                      <a-button 
                        v-permission="'bug:create'"
                        type="primary" 
                        @click="handleCreate"
                      >
                        <template #icon><PlusOutlined /></template>
                        新增Bug
                      </a-button>
                    </a-space>
                  </a-space>
                </template>
                <a-form v-show="searchFormVisible" :model="searchForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
                  <a-row :gutter="16">
                    <a-col :span="6">
                      <a-form-item label="关键词">
                        <a-input
                          v-model:value="searchForm.keyword"
                          placeholder="Bug标题/描述"
                          allow-clear
                        />
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="项目">
                        <a-select
                          v-model:value="searchForm.project_id"
                          placeholder="选择项目"
                          allow-clear
                          show-search
                          :filter-option="filterProjectOption"
                          @change="handleSearchProjectChange"
                        >
                          <a-select-option
                            v-for="project in projects"
                            :key="project.id"
                            :value="project.id"
                          >
                            {{ project.name }}
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="状态">
                        <a-select
                          v-model:value="searchForm.status"
                          mode="multiple"
                          placeholder="选择状态（可多选）"
                          allow-clear
                          show-search
                          max-tag-count="responsive"
                        >
                          <a-select-option value="active">激活</a-select-option>
                          <a-select-option value="resolved">已解决</a-select-option>
                          <a-select-option value="closed">已关闭</a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="优先级">
                        <a-select
                          v-model:value="searchForm.priority"
                          placeholder="选择优先级"
                          allow-clear
                        >
                          <a-select-option value="low">低</a-select-option>
                          <a-select-option value="medium">中</a-select-option>
                          <a-select-option value="high">高</a-select-option>
                          <a-select-option value="urgent">紧急</a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                  </a-row>
                  <a-row :gutter="16">
                    <a-col :span="6">
                      <a-form-item label="严重程度">
                        <a-select
                          v-model:value="searchForm.severity"
                          placeholder="选择严重程度"
                          allow-clear
                        >
                          <a-select-option value="low">低</a-select-option>
                          <a-select-option value="medium">中</a-select-option>
                          <a-select-option value="high">高</a-select-option>
                          <a-select-option value="critical">严重</a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="更新日期">
                        <a-range-picker
                          v-model:value="updatedDateRange"
                          format="YYYY-MM-DD"
                          style="width: 100%"
                        />
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="版本号">
                        <a-select
                          v-model:value="searchForm.version_id"
                          placeholder="选择版本号"
                          allow-clear
                          show-search
                          :filter-option="filterSearchVersionOption"
                          :loading="searchVersionLoading"
                          @focus="loadVersionsForSearch"
                        >
                          <a-select-option
                            v-for="version in searchVersions"
                            :key="version.id"
                            :value="version.id"
                          >
                            {{ version.version_number }}
                          </a-select-option>
                        </a-select>
                      </a-form-item>
                    </a-col>
                    <a-col :span="6">
                      <a-form-item label="指派给" class="assignee-form-item">
                        <div class="assignee-wrapper">
                          <a-select
                            v-model:value="searchForm.assignee_id"
                            placeholder="选择指派给"
                            allow-clear
                            show-search
                            :filter-option="filterAssigneeOption"
                            :loading="searchProjectMembersLoading"
                            style="width: 100%"
                            class="assignee-select"
                            @change="handleAssigneeChange"
                          >
                            <a-select-option
                              v-for="option in assigneeOptions"
                              :key="option.id"
                              :value="option.id"
                            >
                              {{ option.username }}{{ option.nickname ? `(${option.nickname})` : '' }}
                              <span v-if="option.role" style="color: #999; margin-left: 4px">
                                ({{ getRoleText(option.role) }})
                              </span>
                            </a-select-option>
                          </a-select>
                          <a-checkbox v-model:checked="searchForm.assignToMe" @change="handleAssignToMeChange" class="assign-to-me-checkbox">
                            指派给我
                          </a-checkbox>
                        </div>
                      </a-form-item>
                    </a-col>
                  </a-row>
                  <a-row>
                    <a-col :span="24">
                      <a-form-item :wrapper-col="{ offset: 0 }">
                        <a-space style="justify-content: flex-end; width: 100%">
                          <a-button type="primary" @click="handleSearch">查询</a-button>
                          <a-button @click="handleReset">重置</a-button>
                        </a-space>
                      </a-form-item>
                    </a-col>
                  </a-row>
                </a-form>
              </a-card>

              <a-card :bordered="false" class="table-card">
                <a-table
                  :columns="displayColumns"
                  :data-source="bugs"
                  :loading="loading"
                  :pagination="pagination"
                  :scroll="{ x: 'max-content' }"
                  row-key="id"
                  @change="handleTableChange"
                  :custom-row="(record: Bug) => ({
                    onClick: () => handleView(record),
                    class: 'table-row-clickable'
                  })"
                >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'status'">
                  <a-space>
                    <a-tag :color="getStatusColor(record.status)">
                      {{ getStatusText(record.status) }}
                    </a-tag>
                    <a-tag v-if="record.confirmed" color="green">已确认</a-tag>
                    <a-tag v-else-if="record.status === 'active'" color="orange">未确认</a-tag>
                  </a-space>
                </template>
                <template v-else-if="column.key === 'priority'">
                  <a-tag :color="getPriorityColor(record.priority)">
                    {{ getPriorityText(record.priority) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'severity'">
                  <a-tag :color="getSeverityColor(record.severity)">
                    {{ getSeverityText(record.severity) }}
                  </a-tag>
                </template>
                <template v-else-if="column.key === 'project'">
                  {{ record.project?.name || '-' }}
                </template>
                <template v-else-if="column.key === 'versions'">
                  <a-space v-if="record.versions && record.versions.length > 0" size="small">
                    <a-tag
                      v-for="version in record.versions"
                      :key="version.id"
                      color="blue"
                      style="margin-right: 4px"
                    >
                      {{ version.version_number }}
                    </a-tag>
                  </a-space>
                  <span v-else>-</span>
                </template>
                <template v-else-if="column.key === 'creator'">
                  {{ record.creator ? `${record.creator.username}${record.creator.nickname ? `(${record.creator.nickname})` : ''}` : '-' }}
                </template>
                <template v-else-if="column.key === 'assignees'">
                  <a-tag
                    v-for="assignee in record.assignees || []"
                    :key="assignee.id"
                    style="margin-right: 4px"
                  >
                    {{ assignee.username }}{{ assignee.nickname ? `(${assignee.nickname})` : '' }}
                  </a-tag>
                  <span v-if="!record.assignees || record.assignees.length === 0">-</span>
                </template>
                <template v-else-if="column.key === 'updated_at'">
                  {{ formatDateTime(record.updated_at) }}
                </template>
                <template v-else-if="column.key === 'created_at'">
                  {{ formatDateTime(record.created_at) }}
                </template>
                <template v-else-if="column.key === 'action'">
                  <a-space @click.stop>
                    <a-button 
                      v-permission="'bug:update'"
                      type="link" 
                      size="small" 
                      @click.stop="handleEdit(record)"
                    >
                      编辑
                    </a-button>
                    <a-button 
                      v-permission="'bug:assign'"
                      type="link" 
                      size="small" 
                      @click.stop="handleAssign(record)"
                    >
                      指派
                    </a-button>
                    <a-button
                      v-permission="'bug:update'"
                      v-if="record.status === 'active' && !record.confirmed"
                      type="link"
                      size="small"
                      @click.stop="handleConfirm(record)"
                    >
                      确认
                    </a-button>
                    <a-button
                      v-permission="'bug:update'"
                      v-if="record.status === 'active'"
                      type="link"
                      size="small"
                      @click.stop="handleResolve(record)"
                    >
                      解决
                    </a-button>
                    <a-popconfirm
                      v-permission="'bug:delete'"
                      title="确定要删除这个Bug吗？"
                      @confirm="handleDelete(record.id)"
                    >
                      <a-button type="link" size="small" danger @click.stop>删除</a-button>
                    </a-popconfirm>
                  </a-space>
                </template>
              </template>
            </a-table>
          </a-card>
            </a-tab-pane>
          </a-tabs>
        </div>
      </a-layout-content>
    </a-layout>

    <!-- Bug编辑/创建模态框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      :width="800"
      :mask-closable="false"
      @ok="handleSubmit"
      @cancel="handleCancel"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="Bug标题" name="title">
          <a-input v-model:value="formData.title" placeholder="请输入Bug标题" />
        </a-form-item>
        <a-form-item label="Bug描述" name="description">
          <MarkdownEditor
            ref="descriptionEditorRef"
            v-model="formData.description"
            placeholder="请输入Bug描述（支持Markdown）"
            :rows="8"
            :project-id="formData.project_id || 0"
          />
        </a-form-item>
        <a-form-item label="项目" name="project_id">
          <a-select
            v-model:value="formData.project_id"
            placeholder="选择项目"
            show-search
            :filter-option="filterProjectOption"
            @change="handleFormProjectChange"
          >
            <a-select-option
              v-for="project in projects"
              :key="project.id"
              :value="project.id"
            >
              {{ project.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="关联需求" name="requirement_id">
          <a-select
            v-model:value="formData.requirement_id"
            placeholder="选择关联需求（可选）"
            allow-clear
            show-search
            :filter-option="filterRequirementOption"
            :loading="requirementLoading"
            @focus="loadRequirementsForProject"
          >
            <a-select-option
              v-for="requirement in requirements"
              :key="requirement.id"
              :value="requirement.id"
            >
              {{ requirement.title }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="功能模块" name="module_id">
          <a-select
            v-model:value="formData.module_id"
            placeholder="选择功能模块（可选）"
            allow-clear
            show-search
            :filter-option="filterModuleOption"
            :loading="moduleLoading"
            @focus="loadModulesForProject"
          >
            <a-select-option
              v-for="module in modules"
              :key="module.id"
              :value="module.id"
            >
              {{ module.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-select v-model:value="formData.status">
            <a-select-option value="active">激活</a-select-option>
            <a-select-option value="resolved">已解决</a-select-option>
            <a-select-option value="closed">已关闭</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="优先级" name="priority">
          <a-select v-model:value="formData.priority">
            <a-select-option value="low">低</a-select-option>
            <a-select-option value="medium">中</a-select-option>
            <a-select-option value="high">高</a-select-option>
            <a-select-option value="urgent">紧急</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="严重程度" name="severity">
          <a-select v-model:value="formData.severity">
            <a-select-option value="low">低</a-select-option>
            <a-select-option value="medium">中</a-select-option>
            <a-select-option value="high">高</a-select-option>
            <a-select-option value="critical">严重</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="指派给" name="assignee_ids">
          <ProjectMemberSelect
            v-model="formData.assignee_ids"
            :project-id="formData.project_id"
            :multiple="true"
            placeholder="选择指派给"
            :show-role="true"
          />
        </a-form-item>
        <a-form-item label="所属版本" name="version_ids">
          <VersionSelectWithCreate
            v-model="formData.version_ids"
            v-model:create-version="formData.create_version"
            v-model:version-number="formData.version_number"
            :project-id="formData.project_id"
            :multiple="true"
            placeholder="选择所属版本（必填，至少一个）"
            :required="true"
            @validate="handleVersionValidate"
          />
        </a-form-item>
        <a-form-item label="预估工时" name="estimated_hours">
          <a-input-number
            v-model:value="formData.estimated_hours"
            placeholder="预估工时（小时）"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="实际工时" name="actual_hours">
          <a-input-number
            v-model:value="formData.actual_hours"
            placeholder="实际工时（小时）"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
          <span style="margin-left: 8px; color: #999">更新实际工时会自动创建资源分配（使用第一个分配人）</span>
        </a-form-item>
        <a-form-item label="工作日期" name="work_date" v-if="formData.actual_hours">
          <a-date-picker
            v-model:value="formData.work_date"
            placeholder="选择工作日期（可选）"
            style="width: 100%"
            format="YYYY-MM-DD"
            :getPopupContainer="getPopupContainer"
            :popupStyle="{ zIndex: 2100 }"
          />
          <span style="margin-left: 8px; color: #999">不填则使用今天</span>
        </a-form-item>
        <a-form-item label="附件">
          <AttachmentUpload
            v-if="formData.project_id && formData.project_id > 0"
            :project-id="formData.project_id"
            :model-value="formData.attachment_ids"
            :existing-attachments="bugAttachments"
            @update:modelValue="(value) => { formData.attachment_ids = value }"
            @attachment-deleted="handleAttachmentDeleted"
          />
          <span v-else style="color: #999;">请先选择项目后再上传附件</span>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Bug指派模态框 -->
    <a-modal
      v-model:open="assignModalVisible"
      title="指派Bug"
      :mask-closable="true"
      :z-index="2100"
      @ok="handleAssignSubmit"
      @cancel="handleAssignCancel"
    >
      <a-form
        ref="assignFormRef"
        :model="assignFormData"
        :rules="assignFormRules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="指派给" name="assignee_ids">
          <ProjectMemberSelect
            v-model="assignFormData.assignee_ids"
            :project-id="assignFormData.project_id && assignFormData.project_id > 0 ? assignFormData.project_id : undefined"
            :multiple="true"
            placeholder="选择指派给"
            :show-role="true"
            :get-popup-container="getPopupContainer"
          />
        </a-form-item>
        <a-form-item label="备注" name="comment">
          <a-textarea
            v-model:value="assignFormData.comment"
            placeholder="请输入备注（可选）"
            :rows="4"
          />
        </a-form-item>
        <a-form-item label="状态" name="status">
          <a-select
            v-model:value="assignFormData.status"
            placeholder="选择状态（可选）"
            allow-clear
            :getPopupContainer="getPopupContainer"
            :dropdownStyle="{ zIndex: 2100 }"
          >
            <a-select-option value="active">激活</a-select-option>
            <a-select-option value="resolved">已解决</a-select-option>
            <a-select-option value="closed">已关闭</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Bug状态更新模态框 -->
    <a-modal
      v-model:open="statusModalVisible"
      title="更新Bug状态"
      :width="600"
      :mask-closable="true"
      :z-index="2100"
      @ok="handleStatusSubmit"
      @cancel="handleStatusCancel"
    >
      <a-form
        ref="statusFormRef"
        :model="statusFormData"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="新状态">
          <a-select v-model:value="statusFormData.status" disabled>
            <a-select-option value="active">激活</a-select-option>
            <a-select-option value="resolved">已解决</a-select-option>
            <a-select-option value="closed">已关闭</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="解决方案" name="solution">
          <a-select
            v-model:value="statusFormData.solution"
            placeholder="选择解决方案（可选）"
            allow-clear
            :getPopupContainer="getPopupContainer"
            :dropdownStyle="{ zIndex: 2100 }"
          >
            <a-select-option value="设计如此">设计如此</a-select-option>
            <a-select-option value="重复Bug">重复Bug</a-select-option>
            <a-select-option value="外部原因">外部原因</a-select-option>
            <a-select-option value="已解决">已解决</a-select-option>
            <a-select-option value="无法重现">无法重现</a-select-option>
            <a-select-option value="延期处理">延期处理</a-select-option>
            <a-select-option value="不予解决">不予解决</a-select-option>
            <a-select-option value="转为研发需求">转为研发需求</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="备注" name="solution_note">
          <a-textarea
            v-model:value="statusFormData.solution_note"
            placeholder="请输入备注（可选）"
            :rows="4"
          />
        </a-form-item>
        <a-form-item label="预估工时" name="estimated_hours">
          <a-input-number
            v-model:value="statusFormData.estimated_hours"
            placeholder="预估工时（小时）"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="实际工时" name="actual_hours">
          <a-input-number
            v-model:value="statusFormData.actual_hours"
            placeholder="实际工时（小时）"
            :min="0"
            :precision="2"
            style="width: 100%"
          />
          <span style="margin-left: 8px; color: #999">更新实际工时会自动创建资源分配</span>
        </a-form-item>
        <a-form-item label="工作日期" name="work_date" v-if="statusFormData.actual_hours">
          <a-date-picker
            v-model:value="statusFormData.work_date"
            placeholder="选择工作日期（可选）"
            style="width: 100%"
            format="YYYY-MM-DD"
            :getPopupContainer="getPopupContainer"
            :popupStyle="{ zIndex: 2100 }"
          />
          <span style="margin-left: 8px; color: #999">不填则使用今天</span>
        </a-form-item>
        <a-form-item label="解决版本" name="resolved_version_id">
          <VersionSelectWithCreate
            v-model="statusFormData.resolved_version_id"
            v-model:create-version="statusFormData.create_version"
            v-model:version-number="statusFormData.version_number"
            :project-id="statusFormData.project_id"
            :multiple="false"
            placeholder="选择版本（可选）"
            :required="false"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Bug详情弹窗 -->
    <a-modal
      v-model:open="detailModalVisible"
      :width="1200"
      :mask-closable="true"
      :footer="null"
      :z-index="2000"
      @cancel="handleDetailCancel"
    >
      <template #title>
        <div style="width: 100%;">
          <div style="text-align: center;">Bug详情</div>
          <div v-if="detailBug" style="font-size: 14px; color: #666; margin-top: 4px; text-align: left;">
            {{ detailBug.title }}
          </div>
        </div>
      </template>
      <div v-if="detailBug" style="max-height: 70vh; overflow-y: auto">
        <!-- 操作按钮 -->
        <div style="margin-bottom: 16px; display: flex; justify-content: space-between; align-items: center">
          <a-space>
            <a-button
              :disabled="!prevBugId || bugListLoading"
              @click="handleNavigateToPrev"
            >
              ← 上一个
            </a-button>
            <a-button
              :disabled="!nextBugId || bugListLoading"
              @click="handleNavigateToNext"
            >
              下一个 →
            </a-button>
          </a-space>
          <a-space>
            <a-button @click="handleDetailEdit">编辑</a-button>
            <a-button @click="handleDetailAssign">指派</a-button>
            <a-button
              v-if="detailBug.status === 'active' && !detailBug.confirmed"
              @click="handleDetailConfirm"
            >
              确认
            </a-button>
            <a-button
              v-if="detailBug.status === 'active'"
              @click="handleDetailResolve"
            >
              解决
            </a-button>
            <a-button
              @click="handleDetailClose"
              :disabled="detailBug.status !== 'resolved'"
            >
              关闭
            </a-button>
            <a-button
              v-if="detailBug.status === 'active'"
              @click="handleDetailConvertToRequirement"
            >
              转需求
            </a-button>
            <a-popconfirm
              title="确定要删除这个Bug吗？"
              @confirm="handleDetailDelete"
            >
              <a-button danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </div>

        <BugDetailContent
          :bug="detailBug"
          :loading="detailLoading"
          @refresh="handleDetailRefresh"
          @requirement-click="handleDetailRequirementClick"
          @version-click="handleDetailVersionClick"
        />
      </div>
    </a-modal>

    <!-- 列设置弹窗 -->
    <a-modal
      v-model:open="columnSettingsModalVisible"
      title="列设置"
      :width="500"
      :mask-closable="false"
      @ok="handleSaveColumnSettings"
      @cancel="handleCancelColumnSettings"
    >
      <div class="column-settings-list">
        <div
          v-for="(col, index) in columnSettingsList"
          :key="col.key"
          :draggable="true"
          @dragstart="handleDragStart($event, index)"
          @dragover.prevent
          @drop="handleDrop($event, index)"
          class="column-setting-item"
          :class="{ 'dragging': draggedIndex === index }"
        >
          <a-space style="width: 100%; justify-content: space-between">
            <a-space>
              <span class="drag-handle">⋮⋮</span>
              <a-checkbox v-model:checked="col.visible" @change="updateColumnOrder">
                {{ col.title }}
              </a-checkbox>
            </a-space>
          </a-space>
        </div>
      </div>
      <template #footer>
        <a-space>
          <a-button @click="handleResetColumnSettings">重置</a-button>
          <a-button @click="handleCancelColumnSettings">取消</a-button>
          <a-button type="primary" @click="handleSaveColumnSettings">保存</a-button>
        </a-space>
      </template>
    </a-modal>

  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch, nextTick, computed } from 'vue'
import { saveLastSelected, getLastSelected } from '@/utils/storage'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { type Dayjs } from 'dayjs'
import { formatDateTime } from '@/utils/date'
import { PlusOutlined, UpOutlined, DownOutlined, SettingOutlined } from '@ant-design/icons-vue'
import AppHeader from '@/components/AppHeader.vue'
import BugDetailContent from '@/components/BugDetailContent.vue'
import MarkdownEditor from '@/components/MarkdownEditor.vue'
import AttachmentUpload from '@/components/AttachmentUpload.vue'
import ProjectMemberSelect from '@/components/ProjectMemberSelect.vue'
import VersionSelectWithCreate from '@/components/VersionSelectWithCreate.vue'
import { useAuthStore } from '@/stores/auth'
import {
  getBugs,
  getBug,
  createBug,
  updateBug,
  deleteBug,
  updateBugStatus,
  assignBug,
  confirmBug,
  getBugStatistics,
  getBugColumnSettings,
  saveBugColumnSettings,
  type Bug,
  type BugStatus,
  type CreateBugRequest,
  type BugStatistics,
  type ColumnSetting
} from '@/api/bug'
import { getProjects, getProjectMembers, type Project, type ProjectMember } from '@/api/project'
import { getUsers, type User } from '@/api/user'
import { getRequirements, createRequirement, type Requirement, type CreateRequirementRequest } from '@/api/requirement'
import { getModules, type Module } from '@/api/module'
import { getVersions, createVersion, type Version } from '@/api/version'
import { getAttachments, attachToEntity, uploadFile, type Attachment } from '@/api/attachment'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const bugs = ref<Bug[]>([])
const projects = ref<Project[]>([])
const users = ref<User[]>([])
const searchProjectMembers = ref<ProjectMember[]>([]) // 搜索表单中的项目成员列表
const searchProjectMembersLoading = ref(false)
const requirements = ref<Requirement[]>([])
const requirementLoading = ref(false)
const modules = ref<Module[]>([])
const moduleLoading = ref(false)
const searchVersions = ref<Version[]>([])  // 搜索表单中的版本列表
const searchVersionLoading = ref(false)
const statistics = ref<BugStatistics | null>(null)
const activeTab = ref<string>('list')
const searchFormVisible = ref(false) // 搜索栏显示/隐藏状态，默认折叠

// 上一个/下一个bug导航
const prevBugId = ref<number | null>(null)
const nextBugId = ref<number | null>(null)
const bugListLoading = ref(false)

// 详情弹窗相关
const detailModalVisible = ref(false)
const detailLoading = ref(false)
const detailBug = ref<Bug | null>(null)
const validBugStatuses: BugStatus[] = ['active', 'resolved', 'closed']

const searchForm = reactive({
  keyword: '',
  project_id: undefined as number | undefined,
  status: [] as BugStatus[] | undefined,
  priority: undefined as string | undefined,
  severity: undefined as string | undefined,
  version_id: undefined as number | undefined,
  assignee_id: undefined as number | undefined,
  assignToMe: false, // 指派给我
  updated_start_date: undefined as string | undefined,
  updated_end_date: undefined as string | undefined
})

// 更新日期范围选择器
const updatedDateRange = ref<[Dayjs, Dayjs] | null>(null)

const normalizeStatusFilter = (status?: string | null | Array<string | null>): BugStatus[] => {
  const values = Array.isArray(status) ? status : status ? [status] : []

  return Array.from(
    new Set(
      values
        .filter((value): value is string => typeof value === 'string' && value.length > 0)
        .flatMap(value => value.split(','))
        .map(value => value.trim())
        .filter((value): value is BugStatus => validBugStatuses.includes(value as BugStatus))
    )
  )
}

const getStatusFilterParam = () => {
  return Array.isArray(searchForm.status) && searchForm.status.length > 0
    ? searchForm.status.join(',')
    : undefined
}

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showTotal: (total: number) => `共 ${total} 条`,
  showSizeChanger: true,
  showQuickJumper: true,
  position: ['topRight'] as const
})

// 所有可用列的默认配置（不包含"操作"列）
const defaultColumnConfig = [
  { key: 'id', title: '编号', dataIndex: 'id', width: 60, visible: true, order: 1 },
  { key: 'title', title: 'Bug标题', dataIndex: 'title', width: 300, ellipsis: true, visible: true, order: 2 },
  { key: 'project', title: '项目', width: 120, visible: true, order: 3 },
  { key: 'versions', title: '版本号', width: 150, visible: true, order: 4 },
  { key: 'status', title: '状态', width: 100, visible: true, order: 5 },
  { key: 'priority', title: '优先级', width: 100, visible: true, order: 6 },
  { key: 'severity', title: '严重程度', width: 100, visible: true, order: 7 },
  { key: 'creator', title: '创建人', width: 150, visible: true, order: 8 },
  { key: 'assignees', title: '指派给', width: 160, visible: true, order: 9 },
  { key: 'updated_at', title: '更新时间', dataIndex: 'updated_at', width: 180, visible: true, order: 10 },
  { key: 'created_at', title: '创建时间', dataIndex: 'created_at', width: 180, visible: true, order: 11 }
]

// "操作"列固定配置
const actionColumn = { title: '操作', key: 'action', width: 300, fixed: 'right' as const }

// 列设置相关状态
const columnSettingsModalVisible = ref(false)
const columnSettingsList = ref<Array<{ key: string; title: string; visible: boolean; order: number; width?: number; dataIndex?: string; ellipsis?: boolean }>>([])
const draggedIndex = ref<number | null>(null)
const originalColumnSettings = ref<ColumnSetting[]>([])

// 计算属性：根据项目ID决定使用项目成员还是所有用户
const assigneeOptions = computed(() => {
  if (searchForm.project_id) {
    // 如果有项目ID，使用项目成员
    return searchProjectMembers.value.map(member => ({
      id: member.user_id,
      username: member.user?.username || '',
      nickname: member.user?.nickname || '',
      role: member.role
    }))
  } else {
    // 如果没有项目ID，使用所有用户
    return users.value.map(user => ({
      id: user.id,
      username: user.username,
      nickname: user.nickname,
      role: undefined
    }))
  }
})

// 计算属性：根据列设置生成表格列
const displayColumns = computed(() => {
  // 获取可见的列并按order排序
  const visibleColumns = columnSettingsList.value
    .filter(col => col.visible)
    .sort((a, b) => a.order - b.order)
    .map(col => {
      const config = defaultColumnConfig.find(c => c.key === col.key)
      // 对于id列，强制使用默认宽度60
      const width = col.key === 'id' ? (config?.width || 60) : (col.width || config?.width)
      return {
        title: col.title,
        key: col.key,
        dataIndex: col.dataIndex || config?.dataIndex,
        width: width,
        ellipsis: col.ellipsis || config?.ellipsis
      }
    })
  
  // 追加固定的"操作"列
  return [...visibleColumns, actionColumn]
})

const modalVisible = ref(false)
const modalTitle = ref('新增Bug')
const formRef = ref()
const descriptionEditorRef = ref<InstanceType<typeof MarkdownEditor> | null>(null)
const formData = reactive<CreateBugRequest & { id?: number; attachment_ids?: number[]; create_version?: boolean; version_number?: string }>({
  title: '',
  description: '',
  status: 'active',
  priority: 'medium',
  severity: 'medium',
  project_id: 0,
  requirement_id: undefined,
  module_id: undefined,
  assignee_ids: [],
  version_ids: [],  // 所属版本ID列表（必填）
  estimated_hours: undefined,
  attachment_ids: [] as number[],
  create_version: false,
  version_number: undefined
})

const bugAttachments = ref<Attachment[]>([]) // Bug附件列表

// 处理附件删除事件
const handleAttachmentDeleted = (attachmentId: number) => {
  // 从bugAttachments中移除已删除的附件
  bugAttachments.value = bugAttachments.value.filter(a => a.id !== attachmentId)
}

// 监听 attachment_ids 的变化
watch(() => formData.attachment_ids, () => {
  // 监听 attachment_ids 的变化
}, { deep: true })

// 处理版本验证触发
const handleVersionValidate = () => {
  // 当创建新版本状态或版本号变化时，清除并重新触发表单验证
  if (formRef.value) {
    // 使用 nextTick 确保在 Vue 的响应式更新完成后再触发验证
    nextTick(() => {
      // 检查是否应该通过验证：如果勾选了创建新版本且版本号已输入，直接清除错误
      if (formData.create_version && formData.version_number?.trim()) {
        // 直接清除验证错误，不需要重新验证
        formRef.value.clearValidate(['version_ids'])
      } else {
        // 否则，清除错误后重新验证
        formRef.value.clearValidate(['version_ids'])
        nextTick(() => {
          formRef.value.validateFields(['version_ids']).catch(() => {
            // 验证失败时忽略错误，只触发验证
          })
        })
      }
    })
  }
}

// 监听 create_version 和 version_number 的变化，触发表单验证（作为备用）
watch([() => formData.create_version, () => formData.version_number], () => {
  handleVersionValidate()
})

const formRules = {
  title: [{ required: true, message: '请输入Bug标题', trigger: 'blur' }],
  description: [{ required: true, message: '请输入Bug描述', trigger: 'blur' }],
  project_id: [{ required: true, message: '请选择项目', trigger: 'change' }],
  priority: [{ required: true, message: '请选择优先级', trigger: 'change' }],
  assignee_ids: [
    {
      required: true,
      message: '请选择指派给',
      trigger: 'change',
      validator: (_rule: any, value: number[]) => {
        if (!value || value.length === 0) {
          return Promise.reject('请选择指派给')
        }
        return Promise.resolve()
      }
    }
  ],
  version_ids: [
    {
      required: false, // 不设置 required，完全依赖自定义验证器
      trigger: ['change', 'blur'], // 支持多种触发方式
      validator: (_rule: any, value: number[]) => {
        // 如果已选择了版本，则通过验证
        if (value && value.length > 0) {
          return Promise.resolve()
        }
        // 如果没有选择版本，检查是否创建新版本
        // 注意：这里直接访问 formData，因为它是 reactive 的，会获取最新值
        if (formData.create_version) {
          // 如果勾选了创建新版本，检查版本号是否已输入
          const versionNumber = formData.version_number?.trim()
          if (versionNumber) {
            // 版本号已输入，允许通过验证
            return Promise.resolve()
          }
          // 版本号未输入，提示输入版本号
          return Promise.reject('请输入版本号')
        }
        // 既没有选择版本，也没有创建新版本
        return Promise.reject('请至少选择一个所属版本或创建新版本')
      }
    }
  ]
}

const assignModalVisible = ref(false)
const assignFormRef = ref()
const assignFormData = reactive({
  bug_id: 0,
  project_id: 0, // 保存当前Bug的项目ID
  assignee_ids: [] as number[],
  status: undefined as string | undefined,
  comment: undefined as string | undefined
})

const statusModalVisible = ref(false)
const statusFormRef = ref()
const statusFormData = reactive({
  bug_id: 0,
  project_id: undefined as number | undefined,
  status: 'active' as string,
  solution: undefined as string | undefined,
  solution_note: undefined as string | undefined,
  estimated_hours: undefined as number | undefined,
  actual_hours: undefined as number | undefined,
  work_date: undefined as Dayjs | undefined,
  resolved_version_id: undefined as number | undefined,
  version_number: undefined as string | undefined,
  create_version: false
})

const assignFormRules = {
  assignee_ids: [{ required: true, message: '请选择指派给', trigger: 'change' }]
}

// 加载列设置
const loadColumnSettings = async () => {
  try {
    const settings = await getBugColumnSettings()
    console.log('从后端加载的列设置:', settings)
    if (settings && settings.length > 0) {
      // 合并用户设置和默认配置，确保所有列都有设置
      const mergedSettings = defaultColumnConfig.map(defaultCol => {
        const userSetting = settings.find(s => s.key === defaultCol.key)
        if (userSetting) {
          // 确保使用后端返回的 visible 值（即使是 false）
          // 对于id列，强制使用默认宽度（因为默认宽度已更新为60）
          const width = defaultCol.key === 'id' ? defaultCol.width : (userSetting.width !== undefined ? userSetting.width : defaultCol.width)
          const result = {
            ...defaultCol,
            visible: userSetting.visible, // 直接使用后端返回的值
            order: userSetting.order,
            width: width
          }
          console.log(`列 ${defaultCol.key}: 后端visible=${userSetting.visible}, 宽度=${width}`)
          return result
        }
        // 如果后端没有该列的设置，使用默认配置
        return defaultCol
      })
      // 按order排序
      mergedSettings.sort((a, b) => a.order - b.order)
      columnSettingsList.value = mergedSettings
      // 保存完整的设置（包括所有列）
      originalColumnSettings.value = mergedSettings.map(col => ({
        key: col.key,
        visible: col.visible,
        order: col.order,
        width: col.width
      }))
      console.log('合并后的列设置:', columnSettingsList.value)
    } else {
      // 使用默认配置
      columnSettingsList.value = [...defaultColumnConfig]
      originalColumnSettings.value = []
    }
  } catch (error) {
    // 加载失败时使用默认配置
    console.error('加载列设置失败:', error)
    columnSettingsList.value = [...defaultColumnConfig]
    originalColumnSettings.value = []
  }
}

// 显示列设置弹窗
const showColumnSettingsModal = () => {
  // 确保列设置已加载
  if (columnSettingsList.value.length === 0) {
    loadColumnSettings()
  }
  columnSettingsModalVisible.value = true
}

// 保存列设置
const handleSaveColumnSettings = async () => {
  try {
    // 更新order值（按当前顺序）
    columnSettingsList.value.forEach((col, index) => {
      col.order = index + 1
    })
    
    // 确保所有默认列都有设置（包括可能被隐藏的列）
    // 合并 columnSettingsList 和 defaultColumnConfig，确保所有列都被保存
    const allSettingsMap = new Map<string, ColumnSetting>()
    
    // 先添加当前列设置列表中的所有列
    columnSettingsList.value.forEach(col => {
      allSettingsMap.set(col.key, {
        key: col.key,
        visible: col.visible,
        order: col.order,
        width: col.width
      })
    })
    
    // 确保所有默认列都有设置（如果某个列不在列表中，使用默认值）
    defaultColumnConfig.forEach(defaultCol => {
      if (!allSettingsMap.has(defaultCol.key)) {
        allSettingsMap.set(defaultCol.key, {
          key: defaultCol.key,
          visible: defaultCol.visible,
          order: defaultCol.order,
          width: defaultCol.width
        })
      }
    })
    
    // 转换为数组并按order排序
    const settingsToSave: ColumnSetting[] = Array.from(allSettingsMap.values())
      .sort((a, b) => a.order - b.order)
    
    // 调试：打印要保存的数据
    console.log('保存列设置:', settingsToSave)
    
    await saveBugColumnSettings(settingsToSave)
    message.success('列设置已保存')
    originalColumnSettings.value = settingsToSave
    columnSettingsModalVisible.value = false
    
    // 保存后重新加载设置以确保同步
    await loadColumnSettings()
  } catch (error: any) {
    message.error(error.message || '保存列设置失败')
  }
}

// 取消列设置
const handleCancelColumnSettings = () => {
  // 恢复原始设置
  if (originalColumnSettings.value.length > 0) {
    const mergedSettings = defaultColumnConfig.map(defaultCol => {
      const userSetting = originalColumnSettings.value.find(s => s.key === defaultCol.key)
      if (userSetting) {
        return {
          ...defaultCol,
          visible: userSetting.visible,
          order: userSetting.order,
          width: userSetting.width || defaultCol.width
        }
      }
      return defaultCol
    })
    mergedSettings.sort((a, b) => a.order - b.order)
    columnSettingsList.value = mergedSettings
  } else {
    columnSettingsList.value = [...defaultColumnConfig]
  }
  columnSettingsModalVisible.value = false
  draggedIndex.value = null
}

// 重置列设置
const handleResetColumnSettings = () => {
  columnSettingsList.value = [...defaultColumnConfig]
  originalColumnSettings.value = []
}

// 拖拽开始
const handleDragStart = (event: DragEvent, index: number) => {
  draggedIndex.value = index
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

// 拖拽放置
const handleDrop = (event: DragEvent, targetIndex: number) => {
  event.preventDefault()
  if (draggedIndex.value === null || draggedIndex.value === targetIndex) {
    draggedIndex.value = null
    return
  }
  
  const draggedItem = columnSettingsList.value[draggedIndex.value]
  if (!draggedItem) {
    draggedIndex.value = null
    return
  }
  
  columnSettingsList.value.splice(draggedIndex.value, 1)
  columnSettingsList.value.splice(targetIndex, 0, draggedItem)
  
  // 更新order值
  updateColumnOrder()
  draggedIndex.value = null
}

// 更新列顺序
const updateColumnOrder = () => {
  columnSettingsList.value.forEach((col, index) => {
    col.order = index + 1
  })
}

// 加载Bug列表
const loadBugs = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.current,
      size: pagination.pageSize
    }
    if (searchForm.keyword) {
      params.keyword = searchForm.keyword
    }
    if (searchForm.project_id) {
      params.project_id = searchForm.project_id
    }
    const statusParam = getStatusFilterParam()
    if (statusParam) {
      params.status = statusParam
    }
    if (searchForm.priority) {
      params.priority = searchForm.priority
    }
    if (searchForm.severity) {
      params.severity = searchForm.severity
    }
    if (searchForm.version_id) {
      params.version_id = searchForm.version_id
    }
    if (searchForm.assignee_id) {
      params.assignee_id = searchForm.assignee_id
    }
    if (searchForm.updated_start_date) {
      params.updated_start_date = searchForm.updated_start_date
    }
    if (searchForm.updated_end_date) {
      params.updated_end_date = searchForm.updated_end_date
    }
    const response = await getBugs(params)
    bugs.value = response.list
    pagination.total = response.total
    // 同步后端返回的分页信息，确保一致性
    if (response.page !== undefined) {
      pagination.current = response.page
    }
    if (response.page_size !== undefined) {
      pagination.pageSize = response.page_size
    }
    // 加载统计信息
    await loadStatistics()
  } catch (error: any) {
    message.error(error.message || '加载Bug列表失败')
  } finally {
    loading.value = false
  }
}

// 加载统计信息
const loadStatistics = async () => {
  try {
    const params: any = {}
    if (searchForm.keyword) {
      params.keyword = searchForm.keyword
    }
    if (searchForm.project_id) {
      params.project_id = searchForm.project_id
    }
    statistics.value = await getBugStatistics(params)
  } catch (error: any) {
    console.error('加载统计信息失败:', error)
  }
}

// 加载项目列表
const loadProjects = async () => {
  try {
    // 获取所有项目（不分页），用于下拉选择器
    const response = await getProjects({ size: 1000 })
    projects.value = response.list || []
  } catch (error: any) {
    console.error('加载项目列表失败:', error)
  }
}

// 加载用户列表
const loadUsers = async () => {
  try {
    const response = await getUsers()
    users.value = response.list || []
  } catch (error: any) {
    console.error('加载用户列表失败:', error)
  }
}

// 加载搜索表单中的项目成员列表
const loadSearchProjectMembers = async (projectId: number | undefined) => {
  if (!projectId) {
    searchProjectMembers.value = []
    searchProjectMembersLoading.value = false
    return
  }
  
  searchProjectMembersLoading.value = true
  try {
    searchProjectMembers.value = await getProjectMembers(projectId)
  } catch (error: any) {
    console.error('加载项目成员失败:', error)
    searchProjectMembers.value = []
  } finally {
    searchProjectMembersLoading.value = false
  }
}

// 加载需求列表（根据项目）
const loadRequirementsForProject = async () => {
  if (!formData.project_id) {
    requirements.value = []
    return
  }
  requirementLoading.value = true
  try {
    const response = await getRequirements({ project_id: formData.project_id })
    requirements.value = response.list || []
  } catch (error: any) {
    console.error('加载需求列表失败:', error)
  } finally {
    requirementLoading.value = false
  }
}

const loadModulesForProject = async () => {
  // 功能模块是系统资源，不需要项目ID
  moduleLoading.value = true
  try {
    modules.value = await getModules()
  } catch (error: any) {
    console.error('加载模块列表失败:', error)
  } finally {
    moduleLoading.value = false
  }
}


// 加载搜索表单中的版本列表（根据项目）
const loadVersionsForSearch = async () => {
  if (!searchForm.project_id) {
    searchVersions.value = []
    return
  }
  searchVersionLoading.value = true
  try {
    const response = await getVersions({ project_id: searchForm.project_id, size: 1000 })
    searchVersions.value = response.list || []
  } catch (error: any) {
    console.error('加载版本列表失败:', error)
  } finally {
    searchVersionLoading.value = false
  }
}

// 版本筛选（搜索表单）
const filterSearchVersionOption = (input: string, option: any) => {
  const version = searchVersions.value.find(v => v.id === option.value)
  if (!version) return false
  const searchText = input.toLowerCase()
  return version.version_number.toLowerCase().includes(searchText)
}

// 指派给筛选函数（支持项目成员和所有用户）
const filterAssigneeOption = (input: string, option: any) => {
  const optionData = assigneeOptions.value.find(o => o.id === option.value)
  if (!optionData) return false
  const searchText = input.toLowerCase()
  return (
    optionData.username.toLowerCase().includes(searchText) ||
    (optionData.nickname && optionData.nickname.toLowerCase().includes(searchText))
  )
}

// 获取角色文本
const getRoleText = (role: string): string => {
  const roleMap: Record<string, string> = {
    owner: '负责人',
    member: '成员',
    viewer: '查看者'
  }
  return roleMap[role] || role
}

// 监听搜索表单中的项目变化，重新加载项目成员
watch(() => searchForm.project_id, (newProjectId, oldProjectId) => {
  // 加载项目成员
  loadSearchProjectMembers(newProjectId)
  
  // 如果项目ID变化，清空指派给选择（避免选择了不在新项目成员中的用户）
  if (newProjectId !== oldProjectId) {
    searchForm.assignee_id = undefined
    searchForm.assignToMe = false
  }
}, { immediate: true })

// 监听项目变化，重新加载需求和版本
watch(() => formData.project_id, () => {
  formData.requirement_id = undefined
  formData.version_ids = []
  // 功能模块是系统资源，不需要清空
  if (formData.project_id) {
    loadRequirementsForProject()
  } else {
    requirements.value = []
    formData.assignee_ids = []
  }
})

// 切换搜索栏显示/隐藏
const toggleSearchForm = () => {
  searchFormVisible.value = !searchFormVisible.value
}

// 统计项点击处理
const handleStatisticClick = (status?: BugStatus, priority?: string, severity?: string) => {
  // 切换到列表标签页
  activeTab.value = 'list'
  
  // 设置筛选条件
  searchForm.status = status ? [status] : []
  
  if (priority) {
    searchForm.priority = priority
  } else {
    searchForm.priority = undefined
  }
  
  if (severity) {
    searchForm.severity = severity
  } else {
    searchForm.severity = undefined
  }
  
  // 展开搜索表单
  searchFormVisible.value = true
  
  // 重置分页并加载Bug列表
  pagination.current = 1
  loadBugs()
}

// 搜索
const handleSearch = () => {
  // 处理日期范围
  if (updatedDateRange.value && updatedDateRange.value.length === 2) {
    searchForm.updated_start_date = updatedDateRange.value[0].format('YYYY-MM-DD')
    searchForm.updated_end_date = updatedDateRange.value[1].format('YYYY-MM-DD')
  } else {
    searchForm.updated_start_date = undefined
    searchForm.updated_end_date = undefined
  }
  pagination.current = 1
  loadBugs()
}

// 指派给我复选框改变
const handleAssignToMeChange = async (e: any) => {
  const checked = e.target.checked
  if (checked && authStore.user) {
    const currentUserId = Number(authStore.user.id)
    // 直接设置 assignee_id 为当前用户ID，不受项目限制
    searchForm.assignee_id = currentUserId
  } else {
    // 取消选中时，清空 assignee_id
    searchForm.assignee_id = undefined
  }
}

// 指派给下拉框改变
const handleAssigneeChange = (value: number | number[] | undefined) => {
  // 组件是单选模式，所以 value 应该是 number | undefined
  const assigneeId = Array.isArray(value) ? value[0] : value
  // 如果清空了选择，同时取消"指派给我"
  if (!assigneeId && searchForm.assignToMe) {
    searchForm.assignToMe = false
  }
  // 如果选择了其他用户，取消"指派给我"
  if (assigneeId && authStore.user && assigneeId !== authStore.user.id && searchForm.assignToMe) {
    searchForm.assignToMe = false
  }
  // 如果选择了自己，选中"指派给我"
  if (assigneeId && authStore.user && assigneeId === authStore.user.id && !searchForm.assignToMe) {
    searchForm.assignToMe = true
  }
}

// 搜索表单项目选择改变
const handleSearchProjectChange = (value: number | undefined) => {
  saveLastSelected('last_selected_bug_project_search', value)
  // 只有在没有选中"指派给我"时才清空指派给的选择
  if (!searchForm.assignToMe) {
    searchForm.assignee_id = undefined
  }
  // 清空版本选择并重新加载版本列表
  searchForm.version_id = undefined
  if (value) {
    loadVersionsForSearch()
  } else {
    searchVersions.value = []
  }
}

// 编辑表单项目选择改变
const handleFormProjectChange = (value: number | undefined) => {
  saveLastSelected('last_selected_bug_project_form', value || 0)
}

// 重置
const handleReset = () => {
  searchForm.keyword = ''
  searchForm.project_id = undefined
  searchForm.status = []
  searchForm.priority = undefined
  searchForm.severity = undefined
  searchForm.version_id = undefined
  searchForm.assignee_id = undefined
  searchForm.assignToMe = false // 重置"指派给我"
  searchForm.updated_start_date = undefined
  searchForm.updated_end_date = undefined
  updatedDateRange.value = null
  searchVersions.value = []
  pagination.current = 1
  // 清除保存的搜索项目选择
  saveLastSelected('last_selected_bug_project_search', undefined)
  loadBugs()
}

// 表格变化
const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadBugs()
}

// 创建
const handleCreate = () => {
  modalTitle.value = '新增Bug'
  formData.id = undefined
  formData.title = ''
  formData.description = ''
  formData.status = 'active'
  formData.priority = 'medium'
  formData.severity = 'medium'
  // 从 localStorage 恢复最后选择的项目
  const lastProjectId = getLastSelected<number>('last_selected_bug_project_form')
  formData.project_id = lastProjectId || 0
  formData.requirement_id = undefined
  formData.module_id = undefined
  formData.assignee_ids = []
  formData.version_ids = []
  formData.estimated_hours = undefined
  formData.actual_hours = undefined
  formData.work_date = undefined
  formData.attachment_ids = []
  formData.create_version = false
  formData.version_number = undefined
  bugAttachments.value = []
  if (formData.project_id) {
  }
  modalVisible.value = true
}

// 编辑
const handleEdit = async (record: Bug) => {
  modalTitle.value = '编辑Bug'
  formData.id = record.id
  formData.title = record.title
  formData.description = record.description || ''
  formData.status = record.status
  formData.priority = record.priority
  formData.severity = record.severity
  formData.project_id = record.project_id
  formData.requirement_id = record.requirement_id
  formData.module_id = record.module_id
  formData.assignee_ids = record.assignees?.map(a => a.id) || []
  formData.version_ids = record.versions?.map(v => v.id) || []
  formData.estimated_hours = record.estimated_hours
  formData.actual_hours = record.actual_hours
  formData.work_date = undefined
  formData.create_version = false
  formData.version_number = undefined
  
  // 加载Bug附件
  try {
    bugAttachments.value = await getAttachments({ bug_id: record.id })
    formData.attachment_ids = bugAttachments.value.map(a => a.id)
  } catch (error: any) {
    console.error('加载附件失败:', error)
    bugAttachments.value = []
    formData.attachment_ids = []
  }
  
  modalVisible.value = true
  if (formData.project_id) {
    loadRequirementsForProject()
  }
}

// 查看详情
const handleView = async (record: Bug) => {
  detailModalVisible.value = true
  await loadBugDetail(record.id)
}

// 提交
const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    
    // 获取最新的描述内容
    // 优先使用 formData.description（v-model 已经同步），如果有项目ID且有本地图片，则上传并替换
    let description = formData.description || ''
    
    // 如果有项目ID，尝试上传本地图片（如果有的话）
    if (descriptionEditorRef.value && formData.project_id) {
      try {
        // uploadLocalImages 会返回最新的编辑器内容（即使没有本地图片也会返回当前内容）
        const uploadedDescription = await descriptionEditorRef.value.uploadLocalImages(async (file: File, projectId: number) => {
          const attachment = await uploadFile(file, projectId)
          return attachment
        })
        // 使用上传后的内容（可能包含已替换的图片URL）
        description = uploadedDescription
      } catch (error: any) {
        console.error('上传图片失败:', error)
        message.warning('部分图片上传失败，请检查')
        // 上传失败时，使用 formData.description（v-model 已经同步）
        description = formData.description || ''
      }
    }
    
    // 处理版本：如果选择了创建新版本，先创建版本
    let versionIds = [...(formData.version_ids || [])]
    if (formData.create_version) {
      // 检查版本号是否已输入
      const versionNumber = formData.version_number?.trim()
      if (!versionNumber) {
        message.error('请输入版本号')
        return
      }
      try {
        const newVersion = await createVersion({
          version_number: versionNumber,
          project_id: formData.project_id,
          status: 'wait'
        })
        versionIds.push(newVersion.id)
        message.success(`版本 ${newVersion.version_number} 创建成功`)
      } catch (error: any) {
        message.error('创建版本失败: ' + (error.message || '未知错误'))
        return
      }
    }
    
    // 确保至少有一个版本
    if (versionIds.length === 0) {
      message.error('请至少选择一个所属版本或创建新版本')
      return
    }
    
    // 确保 description 字段总是存在（即使是空字符串）
    const data: any = {
      title: formData.title,
      description: description || '', // 确保 description 字段总是存在
      status: formData.status,
      priority: formData.priority,
      severity: formData.severity,
      project_id: formData.project_id,
      requirement_id: formData.requirement_id,
      module_id: formData.module_id,
      assignee_ids: formData.assignee_ids,
      version_ids: versionIds,
      estimated_hours: formData.estimated_hours,
      actual_hours: formData.actual_hours,
      work_date: formData.work_date && typeof formData.work_date !== 'string' && 'isValid' in formData.work_date && (formData.work_date as Dayjs).isValid() ? (formData.work_date as Dayjs).format('YYYY-MM-DD') : (typeof formData.work_date === 'string' ? formData.work_date : undefined)
    }
    
    // 始终发送 attachment_ids，如果为 undefined 或 null，发送空数组
    // 注意：必须显式设置，不能依赖对象字面量，因为 undefined 值会被 JSON 序列化忽略
    const attachmentIdsValue = formData.attachment_ids
    if (attachmentIdsValue === undefined || attachmentIdsValue === null) {
      data.attachment_ids = []
    } else {
      data.attachment_ids = Array.isArray(attachmentIdsValue) ? attachmentIdsValue : []
    }
    
    let bugId: number
    if (formData.id) {
      bugId = formData.id
      await updateBug(bugId, data)
      message.success('更新成功')
    } else {
      const newBug = await createBug(data)
      bugId = newBug.id
      message.success('创建成功')
      
      // 创建Bug后，如果有待上传的附件，需要关联到Bug
      if (formData.attachment_ids && formData.attachment_ids.length > 0 && formData.project_id) {
        try {
          for (const attachmentId of formData.attachment_ids) {
            await attachToEntity(attachmentId, { bug_id: bugId })
          }
        } catch (error: any) {
          console.error('关联附件到Bug失败:', error)
        }
      }
    }
    modalVisible.value = false
    loadBugs()
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '操作失败')
  }
}

// 取消
const handleCancel = () => {
  formRef.value?.resetFields()
  requirements.value = []
}

// 删除
const handleDelete = async (id: number) => {
  try {
    await deleteBug(id)
    message.success('删除成功')
    loadBugs()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

// 解决Bug（弹出对话框）
const handleResolve = (record: Bug) => {
  handleOpenStatusModal(record, 'resolved')
}

// 状态变更（保留用于其他场景，如详情页）
// @ts-ignore
const _handleStatusChange = (record: Bug, status: string) => {
  // 只有"已解决"状态才弹出对话框
  if (status === 'resolved') {
    handleOpenStatusModal(record, status)
  } else {
    // 其他状态直接更新
    handleStatusUpdate(record.id, status)
  }
}

// 直接更新状态（不弹对话框）
const handleStatusUpdate = async (id: number, status: string) => {
  try {
    await updateBugStatus(id, { status: status as any })
    message.success('状态更新成功')
    loadBugs()
  } catch (error: any) {
    message.error(error.message || '状态更新失败')
  }
}

// 打开状态更新对话框（仅用于"已解决"状态）
const handleOpenStatusModal = (record: Bug, status: string) => {
  statusFormData.bug_id = record.id
  statusFormData.project_id = record.project_id
  statusFormData.status = status
  statusFormData.solution = record.solution
  statusFormData.solution_note = record.solution_note
  statusFormData.estimated_hours = record.estimated_hours
  statusFormData.actual_hours = record.actual_hours
  statusFormData.work_date = undefined
  statusFormData.resolved_version_id = record.resolved_version_id
  statusFormData.version_number = undefined
  statusFormData.create_version = false
  statusModalVisible.value = true
}


// 状态更新提交
const handleStatusSubmit = async () => {
  try {
    const data: any = {
      status: statusFormData.status
    }
    if (statusFormData.solution) {
      data.solution = statusFormData.solution
    }
    if (statusFormData.solution_note) {
      data.solution_note = statusFormData.solution_note
    }
    if (statusFormData.estimated_hours !== undefined) {
      data.estimated_hours = statusFormData.estimated_hours
    }
    if (statusFormData.actual_hours !== undefined) {
      data.actual_hours = statusFormData.actual_hours
      if (statusFormData.work_date && statusFormData.work_date.isValid()) {
        data.work_date = statusFormData.work_date.format('YYYY-MM-DD')
      }
    }
    if (statusFormData.create_version) {
      // 检查版本号是否已输入
      const versionNumber = statusFormData.version_number?.trim()
      if (!versionNumber) {
        message.error('请输入版本号')
        return
      }
      data.create_version = true
      data.version_number = versionNumber
    } else if (statusFormData.resolved_version_id) {
      data.resolved_version_id = statusFormData.resolved_version_id
    }
    await updateBugStatus(statusFormData.bug_id, data)
    message.success('状态更新成功')
    statusModalVisible.value = false
    loadBugs()
  } catch (error: any) {
    message.error(error.message || '状态更新失败')
  }
}

// 状态更新取消
const handleStatusCancel = () => {
  statusModalVisible.value = false
  statusFormData.bug_id = 0
  statusFormData.project_id = undefined
  statusFormData.status = 'active'
  statusFormData.solution = undefined
  statusFormData.solution_note = undefined
  statusFormData.estimated_hours = undefined
  statusFormData.actual_hours = undefined
  statusFormData.work_date = undefined
  statusFormData.resolved_version_id = undefined
  statusFormData.version_number = undefined
  statusFormData.create_version = false
}


// 获取下拉框容器（用于解决模态框中下拉框被遮挡的问题）
const getPopupContainer = (triggerNode: HTMLElement): HTMLElement => {
  return triggerNode.parentElement || document.body
}

// 指派
const handleAssign = (record: Bug) => {
  assignFormData.bug_id = record.id
  // 确保 project_id 是有效的（优先使用 project?.id，如果不存在则使用 project_id）
  const projectId = record.project?.id || record.project_id
  assignFormData.project_id = projectId && projectId > 0 ? projectId : 0
  assignFormData.assignee_ids = [] // 默认清空，不预填充当前值
  assignFormData.status = record.status // 设置当前Bug的状态
  assignFormData.comment = undefined // 清空备注
  assignModalVisible.value = true
}

// 指派提交
const handleAssignSubmit = async () => {
  try {
    await assignFormRef.value.validate()
    const requestData: any = { assignee_ids: assignFormData.assignee_ids }
    if (assignFormData.status) {
      requestData.status = assignFormData.status
    }
    if (assignFormData.comment) {
      requestData.comment = assignFormData.comment
    }
    await assignBug(assignFormData.bug_id, requestData)
    message.success('指派成功')
    assignModalVisible.value = false
    loadBugs()
  } catch (error: any) {
    if (error.errorFields) {
      return
    }
    message.error(error.message || '指派失败')
  }
}

// 指派取消
const handleAssignCancel = () => {
  assignFormRef.value?.resetFields()
  assignFormData.status = undefined
  assignFormData.comment = undefined
  assignFormData.assignee_ids = []
}

// 确认Bug
const handleConfirm = async (record: Bug) => {
  try {
    await confirmBug(record.id)
    message.success('确认成功')
    loadBugs()
  } catch (error: any) {
    message.error(error.message || '确认失败')
  }
}

// 获取状态颜色
const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    active: 'orange',
    resolved: 'green',
    closed: 'default'
  }
  return colors[status] || 'default'
}

// 获取状态文本
const getStatusText = (status: string | undefined) => {
  if (!status) return '-'
  const texts: Record<string, string> = {
    active: '激活',
    resolved: '已解决',
    closed: '已关闭'
  }
  return texts[status.toLowerCase()] || status
}

// 获取优先级颜色
const getPriorityColor = (priority: string) => {
  const colors: Record<string, string> = {
    low: 'default',
    medium: 'blue',
    high: 'orange',
    urgent: 'red'
  }
  return colors[priority] || 'default'
}

// 获取优先级文本
const getPriorityText = (priority: string) => {
  const texts: Record<string, string> = {
    low: '低',
    medium: '中',
    high: '高',
    urgent: '紧急'
  }
  return texts[priority] || priority
}

// 获取严重程度颜色
const getSeverityColor = (severity: string) => {
  const colors: Record<string, string> = {
    low: 'default',
    medium: 'blue',
    high: 'orange',
    critical: 'red'
  }
  return colors[severity] || 'default'
}

// 获取严重程度文本
const getSeverityText = (severity: string) => {
  const texts: Record<string, string> = {
    low: '低',
    medium: '中',
    high: '高',
    critical: '严重'
  }
  return texts[severity] || severity
}

// 项目筛选
const filterProjectOption = (input: string, option: any) => {
  const project = projects.value.find(p => p.id === option.value)
  if (!project) return false
  const searchText = input.toLowerCase()
  return (
    project.name.toLowerCase().includes(searchText) ||
    (project.code && project.code.toLowerCase().includes(searchText))
  )
}

// 需求筛选
const filterRequirementOption = (input: string, option: any) => {
  const requirement = requirements.value.find(r => r.id === option.value)
  if (!requirement) return false
  const searchText = input.toLowerCase()
  return requirement.title.toLowerCase().includes(searchText)
}

const filterModuleOption = (input: string, option: any) => {
  const module = modules.value.find(m => m.id === option.value)
  if (!module) return false
  const searchText = input.toLowerCase()
  return module.name.toLowerCase().includes(searchText) ||
    (module.code && module.code.toLowerCase().includes(searchText))
}

// 加载Bug详情
const loadBugDetail = async (bugId: number) => {
  detailLoading.value = true
  try {
    detailBug.value = await getBug(bugId)
    // 加载相邻bug信息
    await loadAdjacentBugs(bugId)
  } catch (error: any) {
    message.error(error.message || '加载Bug详情失败')
    detailModalVisible.value = false
  } finally {
    detailLoading.value = false
  }
}

// 处理详情刷新事件
const handleDetailRefresh = async () => {
  if (detailBug.value?.id) {
    await loadBugDetail(detailBug.value.id)
    loadBugs() // 同时刷新列表
  }
}

// 处理需求点击事件
const handleDetailRequirementClick = (requirementId: number) => {
  detailModalVisible.value = false
  router.push(`/requirement/${requirementId}`)
}

// 处理版本点击事件
const handleDetailVersionClick = (versionId: number) => {
  detailModalVisible.value = false
  router.push(`/version/${versionId}`)
}

// 加载相邻bug（上一个和下一个）
const loadAdjacentBugs = async (currentBugId: number) => {
  if (!currentBugId) return
  
  bugListLoading.value = true
  prevBugId.value = null
  nextBugId.value = null
  
  try {
    // 使用与当前列表相同的筛选条件
    const baseParams: any = {}
    if (searchForm.keyword) {
      baseParams.keyword = searchForm.keyword
    }
    if (searchForm.project_id) {
      baseParams.project_id = searchForm.project_id
    }
    const statusParam = getStatusFilterParam()
    if (statusParam) {
      baseParams.status = statusParam
    }
    if (searchForm.priority) {
      baseParams.priority = searchForm.priority
    }
    if (searchForm.severity) {
      baseParams.severity = searchForm.severity
    }
    if (searchForm.assignee_id) {
      baseParams.assignee_id = searchForm.assignee_id
    }
    
    // 先获取总数，确定需要查询多少页
    const totalParams = { ...baseParams, page: 1, size: 100 }
    const totalResponse = await getBugs(totalParams)
    const total = totalResponse.total || 0
    const pageSize = 100 // 后端最大限制
    
    if (total === 0) {
      return
    }
    
    // 通过分页查询找到当前bug所在的页
    let currentPage = -1
    let currentIndex = -1
    const maxPages = Math.ceil(total / pageSize)
    
    // 线性查找当前bug所在的页
    for (let page = 1; page <= maxPages; page++) {
      const params = { ...baseParams, page, size: pageSize }
      const response = await getBugs(params)
      const bugs = response.list || []
      const index = bugs.findIndex(b => b.id === currentBugId)
      if (index !== -1) {
        currentPage = page
        currentIndex = index
        break
      }
    }
    
    if (currentPage === -1 || currentIndex === -1) {
      return
    }
    
    // 获取当前页的bug列表
    const currentPageParams = { ...baseParams, page: currentPage, size: pageSize }
    const currentPageResponse = await getBugs(currentPageParams)
    const currentPageBugs = currentPageResponse.list || []
    
    // 获取上一个bug
    if (currentIndex > 0) {
      // 在当前页的前一个
      const prevBug = currentPageBugs[currentIndex - 1]
      if (prevBug) {
        prevBugId.value = prevBug.id
      }
    } else if (currentPage > 1) {
      // 在前一页的最后一个
      const prevPageParams = { ...baseParams, page: currentPage - 1, size: pageSize }
      const prevPageResponse = await getBugs(prevPageParams)
      const prevPageBugs = prevPageResponse.list || []
      if (prevPageBugs.length > 0) {
        const lastBug = prevPageBugs[prevPageBugs.length - 1]
        if (lastBug) {
          prevBugId.value = lastBug.id
        }
      }
    }
    
    // 获取下一个bug
    if (currentIndex < currentPageBugs.length - 1) {
      // 在当前页的下一个
      const nextBug = currentPageBugs[currentIndex + 1]
      if (nextBug) {
        nextBugId.value = nextBug.id
      }
    } else if (currentPage < maxPages) {
      // 在下一页的第一个
      const nextPageParams = { ...baseParams, page: currentPage + 1, size: pageSize }
      const nextPageResponse = await getBugs(nextPageParams)
      const nextPageBugs = nextPageResponse.list || []
      if (nextPageBugs.length > 0) {
        const firstBug = nextPageBugs[0]
        if (firstBug) {
          nextBugId.value = firstBug.id
        }
      }
    }
  } catch (error: any) {
    console.error('加载相邻bug失败:', error)
  } finally {
    bugListLoading.value = false
  }
}

// 导航到上一个bug
const handleNavigateToPrev = () => {
  if (!prevBugId.value) {
    message.warning('没有上一个bug')
    return
  }
  loadBugDetail(prevBugId.value)
}

// 导航到下一个bug
const handleNavigateToNext = () => {
  if (!nextBugId.value) {
    message.warning('没有下一个bug')
    return
  }
  loadBugDetail(nextBugId.value)
}

// 详情弹窗取消
const handleDetailCancel = () => {
  detailBug.value = null
}

// 详情页指派
const handleDetailAssign = () => {
  if (!detailBug.value) return
  handleAssign(detailBug.value)
}

// 详情页确认
const handleDetailConfirm = async () => {
  if (!detailBug.value) return
  try {
    await confirmBug(detailBug.value.id)
    message.success('确认成功')
    await loadBugDetail(detailBug.value.id)
    loadBugs() // 刷新列表
  } catch (error: any) {
    message.error(error.message || '确认失败')
  }
}

// 详情页解决
const handleDetailResolve = () => {
  if (!detailBug.value) return
  handleResolve(detailBug.value)
}

// 详情页关闭
const handleDetailClose = async () => {
  if (!detailBug.value) return
  
  if (detailBug.value.status !== 'resolved') {
    message.warning('只有已解决的Bug才能关闭')
    return
  }
  
  try {
    await updateBugStatus(detailBug.value.id, { status: 'closed' })
    message.success('关闭成功')
    await loadBugDetail(detailBug.value.id)
    loadBugs() // 刷新列表
  } catch (error: any) {
    message.error(error.message || '关闭失败')
  }
}

// 详情页Bug转需求
const handleDetailConvertToRequirement = async () => {
  if (!detailBug.value) return
  
  const confirmed = await new Promise<boolean>((resolve) => {
    const modal = Modal.confirm({
      title: '确认转换',
      content: '确定要将此Bug转为需求吗？转换后将创建新需求，并将Bug状态更新为"已解决"。',
      okText: '确定',
      cancelText: '取消',
      zIndex: 2100,
      onOk: () => {
        resolve(true)
        modal.destroy()
      },
      onCancel: () => {
        resolve(false)
        modal.destroy()
      }
    })
  })
  
  if (!confirmed) return
  
  try {
    const requirementData: CreateRequirementRequest = {
      title: `[转需求] ${detailBug.value.title}`,
      description: detailBug.value.description 
        ? `${detailBug.value.description}\n\n---\n\n*由Bug #${detailBug.value.id}转换而来*`
        : `*由Bug #${detailBug.value.id}转换而来*`,
      project_id: detailBug.value.project_id,
      priority: detailBug.value.priority,
      status: 'draft',
      assignee_id: detailBug.value.assignees && detailBug.value.assignees.length > 0 
        ? detailBug.value.assignees[0].id 
        : undefined,
      estimated_hours: detailBug.value.estimated_hours
    }
    
    const requirement = await createRequirement(requirementData)
    
    await updateBugStatus(detailBug.value.id, {
      status: 'resolved',
      solution: '转为研发需求',
      solution_note: `已转为需求 #${requirement.id}`
    })
    
    await updateBug(detailBug.value.id, {
      requirement_id: requirement.id
    })
    
    message.success(`转换成功，已创建需求 #${requirement.id}`)
    await loadBugDetail(detailBug.value.id)
    loadBugs() // 刷新列表
  } catch (error: any) {
    message.error(error.message || '转换失败')
  }
}

// 详情页删除
const handleDetailDelete = async () => {
  if (!detailBug.value) return
  try {
    await deleteBug(detailBug.value.id)
    message.success('删除成功')
    detailModalVisible.value = false
    loadBugs()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}


// 监听 tab 切换，切换到统计 tab 时加载统计信息
watch(activeTab, (newTab) => {
  if (newTab === 'statistics') {
    loadStatistics()
  }
})

// 记录详情弹窗是否应该保持打开
const shouldKeepDetailOpen = ref(false)

// 详情页编辑
const handleDetailEdit = async () => {
  if (!detailBug.value) return
  shouldKeepDetailOpen.value = true
  detailModalVisible.value = false // 先关闭详情弹窗
  await nextTick() // 等待弹窗关闭
  handleEdit(detailBug.value)
}

// 监听编辑/指派/解决等操作完成，刷新详情
watch([modalVisible, assignModalVisible, statusModalVisible], ([editVisible, assignVisible, statusVisible], [prevEditVisible, prevAssignVisible, prevStatusVisible]) => {
  // 当模态框从打开变为关闭时
  if (prevEditVisible && !editVisible && shouldKeepDetailOpen.value && detailBug.value) {
    shouldKeepDetailOpen.value = false
    // 操作完成后重新打开详情弹窗并刷新
    nextTick(() => {
      detailModalVisible.value = true
      loadBugDetail(detailBug.value!.id)
      loadBugs() // 同时刷新列表
    })
  } else if (!editVisible && !assignVisible && !statusVisible && detailModalVisible.value && detailBug.value) {
    // 其他操作（指派、解决）完成后刷新详情
    loadBugDetail(detailBug.value.id)
    loadBugs() // 同时刷新列表
  }
})

onMounted(async () => {
  // 加载列设置
  await loadColumnSettings()
  // 先加载项目列表，确保项目选择器有数据
  await loadProjects()
  loadUsers()
  
  // 读取路由查询参数（优先级高于 localStorage）
  if (route.query.project_id) {
    searchForm.project_id = Number(route.query.project_id)
  } else {
    // 从 localStorage 恢复最后选择的搜索项目
    const lastSearchProjectId = getLastSelected<number>('last_selected_bug_project_search')
    if (lastSearchProjectId) {
      searchForm.project_id = lastSearchProjectId
    }
  }
  
  // 读取路由查询参数
  searchForm.status = normalizeStatusFilter(route.query.status)
  if (route.query.assignee === 'me' && authStore.user) {
    searchForm.assignToMe = true
    searchForm.assignee_id = authStore.user.id
  }
  
  // 使用 nextTick 确保项目列表已渲染后再加载Bug
  nextTick(() => {
    loadBugs()
  })
})
</script>

<style scoped>
/* 列设置相关样式 */
.column-settings-list {
  max-height: 400px;
  overflow-y: auto;
}

.column-setting-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  cursor: move;
  background: #fff;
  transition: all 0.3s;
}

.column-setting-item:hover {
  border-color: #40a9ff;
  background: #f0f7ff;
}

.column-setting-item.dragging {
  opacity: 0.5;
}

.drag-handle {
  color: #999;
  cursor: move;
  user-select: none;
  margin-right: 8px;
}

.drag-handle:hover {
  color: #40a9ff;
}
.bug-management {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.bug-management :deep(.ant-layout) {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content {
  padding: 24px;
  background: #f0f2f5;
  flex: 1;
  height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.content-inner {
  background: white;
  padding: 24px;
  border-radius: 4px;
  max-width: 100%;
  margin: 0 auto;
  width: 100%;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  height: 0;
}

/* 减小搜索栏卡片底部内边距 */
.content-inner :deep(.ant-tabs-tabpane) > .ant-card:first-child {
  margin-bottom: 0;
}

.content-inner :deep(.ant-tabs-tabpane) > .ant-card:first-child .ant-card-body {
  padding-bottom: 0;
}

/* 统计卡片可点击样式 */
.statistic-card-clickable {
  cursor: pointer;
  transition: all 0.3s;
}

.statistic-card-clickable:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: translateY(-2px);
}

/* 嵌套卡片样式优化 */
.statistic-card-clickable :deep(.ant-card-body) {
  padding: 16px;
}

.table-card {
  margin-top: 0;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-card :deep(.ant-card-body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 8px 16px 16px 16px;
}

.table-card :deep(.ant-table-wrapper) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-card :deep(.ant-spin-nested-loading) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-card :deep(.ant-spin-container) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0;
}

.table-card :deep(.ant-table) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.table-card :deep(.ant-table-container) {
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

/* Tab 相关样式，避免水平滚动条 */
.content-inner :deep(.ant-tabs-tabpane) {
  overflow-x: hidden;
  max-width: 100%;
  box-sizing: border-box;
}

.content-inner :deep(.ant-tabs-tabpane) .ant-row {
  margin-left: 0 !important;
  margin-right: 0 !important;
  max-width: 100%;
}

.content-inner :deep(.ant-tabs-tabpane) .ant-col {
  padding-left: 8px;
  padding-right: 8px;
  max-width: 100%;
  box-sizing: border-box;
}

.content-inner :deep(.ant-tabs-tabpane) .ant-card {
  max-width: 100%;
  box-sizing: border-box;
}

/* 限制表格行高为单行文本 */
.table-card :deep(.ant-table-tbody > tr > td) {
  padding-top: 8px;
  padding-bottom: 8px;
  height: auto;
  max-height: 32px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  line-height: 16px;
}

/* 详情弹窗样式 */
.markdown-content {
  min-height: 200px;
}

/* 详情弹窗样式 */
.markdown-content {
  min-height: 200px;
}

.table-card :deep(.ant-table-tbody > tr > td > div) {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

/* 表格行可点击样式 */
.table-card :deep(.ant-table-tbody > tr.table-row-clickable) {
  cursor: pointer;
}

.table-card :deep(.ant-table-tbody > tr.table-row-clickable:hover) {
  background-color: #f5f5f5;
}

/* Bug标题列宽度固定为300px */
/* 固定编号列（第一列）宽度为60px */
.table-card :deep(.ant-table-thead > tr > th:first-child),
.table-card :deep(.ant-table-tbody > tr > td:first-child) {
  width: 60px !important;
  min-width: 60px !important;
  max-width: 60px !important;
}

/* Bug标题列（第二列）宽度固定为300px */
.table-card :deep(.ant-table-thead > tr > th:nth-child(2)),
.table-card :deep(.ant-table-tbody > tr > td:nth-child(2)) {
  width: 300px !important;
  min-width: 300px !important;
  max-width: 300px !important;
}

/* 指派给字段对齐样式 */
.assignee-form-item {
  margin-bottom: 0;
}

.assignee-form-item :deep(.ant-form-item-control) {
  margin: 0;
  padding: 0;
}

.assignee-form-item :deep(.ant-form-item-control-input) {
  margin: 0;
  padding: 0;
}

.assignee-form-item :deep(.ant-form-item-control-input-content) {
  margin: 0;
  padding: 0;
  line-height: 32px;
}

.assignee-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  margin: 0;
  padding: 0;
}

.assignee-select {
  flex: 1;
  min-width: 0;
  margin: 0;
}

.assign-to-me-checkbox {
  flex-shrink: 0;
  margin: 0;
  white-space: nowrap;
}
</style>
