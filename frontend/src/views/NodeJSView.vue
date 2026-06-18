<script setup lang="ts">
import { onMounted, ref } from 'vue'
import api from '@/api'
import { ElMessage, ElMessageBox } from 'element-plus'

const projects = ref<any[]>([])
const dialogVisible = ref(false)
const form = ref({ name: '', path: '', port: 3000, node_ver: '20' })

async function load() {
  const res: any = await api.get('/nodejs')
  projects.value = res.data || []
}

async function handleCreate() {
  await api.post('/nodejs', form.value)
  ElMessage.success('Node.js 项目已创建')
  dialogVisible.value = false
  load()
}

async function toggle(row: any) {
  const status = row.status === 'running' ? 'stopped' : 'running'
  await api.patch(`/nodejs/${row.id}/toggle`, { status })
  load()
}

async function handleDelete(id: number) {
  await ElMessageBox.confirm('确定删除？', '提示', { type: 'warning' })
  await api.delete(`/nodejs/${id}`)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="page-header">
      <h2>Node.js Project</h2>
      <el-button type="primary" @click="dialogVisible = true">添加项目</el-button>
    </div>
    <el-table :data="projects" stripe>
      <el-table-column prop="name" label="项目名" />
      <el-table-column prop="path" label="路径" />
      <el-table-column prop="port" label="端口" width="80" />
      <el-table-column prop="node_ver" label="Node" width="80" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }"><el-tag :type="row.status === 'running' ? 'success' : 'info'">{{ row.status }}</el-tag></template>
      </el-table-column>
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button text type="primary" @click="toggle(row)">{{ row.status === 'running' ? '停止' : '启动' }}</el-button>
          <el-button text type="danger" @click="handleDelete(row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="dialogVisible" title="添加 Node.js 项目" width="480px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="路径"><el-input v-model="form.path" /></el-form-item>
        <el-form-item label="端口"><el-input-number v-model="form.port" /></el-form-item>
        <el-form-item label="Node"><el-select v-model="form.node_ver"><el-option label="20" value="20" /><el-option label="18" value="18" /></el-select></el-form-item>
      </el-form>
      <template #footer><el-button @click="dialogVisible = false">取消</el-button><el-button type="primary" @click="handleCreate">创建</el-button></template>
    </el-dialog>
  </div>
</template>
