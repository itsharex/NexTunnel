<template>
  <div class="page-stack">
    <n-card
      class="panel acl-panel"
      :bordered="false"
    >
      <template #header>
        <div class="panel-header">
          <div>
            <p class="eyebrow">
              访问控制
            </p>
            <h2>ACL 规则</h2>
          </div>
          <n-tag
            round
            size="small"
            :bordered="false"
          >
            {{ dashboard.snapshot.aclRules.length }} 条规则
          </n-tag>
        </div>
      </template>

      <n-form
        class="acl-form"
        label-placement="top"
        :show-feedback="false"
        @submit.prevent="handleSubmitACLRule"
      >
        <n-form-item label="来源">
          <n-input
            v-model:value="dashboard.aclForm.source"
            placeholder="node-a 或 *"
          />
        </n-form-item>
        <n-form-item label="目标">
          <n-input
            v-model:value="dashboard.aclForm.target"
            placeholder="node-b 或 10.0.0.0/24"
          />
        </n-form-item>
        <n-form-item label="协议">
          <n-select
            v-model:value="dashboard.aclForm.protocol"
            :options="protocolOptions"
          />
        </n-form-item>
        <n-form-item label="动作">
          <n-select
            v-model:value="dashboard.aclForm.action"
            :options="actionOptions"
          />
        </n-form-item>
        <n-form-item label="优先级">
          <n-input-number
            v-model:value="dashboard.aclForm.priority"
            :min="0"
          />
        </n-form-item>
        <div class="form-wide page-actions">
          <n-checkbox v-model:checked="dashboard.aclForm.enabled">
            启用
          </n-checkbox>
          <n-button
            type="primary"
            attr-type="submit"
            :loading="dashboard.isSubmitting"
          >
            添加规则
          </n-button>
        </div>
      </n-form>

      <div class="acl-list">
        <div
          v-for="rule in dashboard.sortedACLRules"
          :key="rule.id"
          class="acl-row"
        >
          <n-tag
            round
            size="small"
            :type="rule.enabled ? 'success' : 'warning'"
            :bordered="false"
          >
            {{ rule.enabled ? '启用' : '停用' }}
          </n-tag>
          <strong>{{ rule.source }} -> {{ rule.target }}</strong>
          <span>{{ rule.protocol.toUpperCase() }} · {{ aclActionLabel(rule.action) }} · P{{ rule.priority }}</span>
          <ConfirmButton
            size="small"
            quaternary
            :loading="dashboard.deletingACLIDs.has(rule.id)"
            :message="`确认删除 ACL 规则 ${rule.id}？`"
            @confirm="handleRemoveACLRule(rule.id)"
          >
            删除
          </ConfirmButton>
        </div>
        <n-empty
          v-if="dashboard.sortedACLRules.length === 0"
          description="暂无 ACL 规则"
        />
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NButton, NCard, NCheckbox, NEmpty, NForm, NFormItem, NInput, NInputNumber, NSelect, NTag, type SelectOption } from 'naive-ui'
import ConfirmButton from '../components/common/ConfirmButton.vue'
import { useAuthStore } from '../stores/auth'
import { useDashboardStore } from '../stores/dashboard'

const protocolOptions: SelectOption[] = [
  { label: 'TCP', value: 'tcp' },
  { label: 'UDP', value: 'udp' },
  { label: 'ICMP', value: 'icmp' },
  { label: 'ANY', value: 'any' },
]

const actionOptions: SelectOption[] = [
  { label: '允许', value: 'allow' },
  { label: '拒绝', value: 'deny' },
]

const auth = useAuthStore()
const dashboard = useDashboardStore()

const aclActionLabel = (action: string): string => (action === 'deny' ? '拒绝' : '允许')

const handleSubmitACLRule = async (): Promise<void> => {
  await dashboard.submitACLRule(auth.token)
}

const handleRemoveACLRule = async (ruleID: string): Promise<void> => {
  await dashboard.removeACLRule(auth.token, ruleID)
}
</script>
