import { describe, it, expect, beforeEach } from 'vitest'
import { useManagementStore } from './managementStore'

describe('managementStore', () => {
  beforeEach(() => {
    useManagementStore.getState().reset()
  })

  describe('activeTab', () => {
    it('defaults to streams', () => {
      expect(useManagementStore.getState().activeTab).toBe('streams')
    })

    it('sets active tab', () => {
      useManagementStore.getState().setActiveTab('kv')
      expect(useManagementStore.getState().activeTab).toBe('kv')
    })
  })

  describe('streams tab', () => {
    it('toggles group expansion', () => {
      expect(useManagementStore.getState().streamsExpandedGroups.regular).toBe(true)
      useManagementStore.getState().toggleStreamsGroup('regular')
      expect(useManagementStore.getState().streamsExpandedGroups.regular).toBe(false)
      useManagementStore.getState().toggleStreamsGroup('regular')
      expect(useManagementStore.getState().streamsExpandedGroups.regular).toBe(true)
    })

    it('sets creating state', () => {
      useManagementStore.getState().setStreamsIsCreating(true)
      expect(useManagementStore.getState().streamsIsCreating).toBe(true)
      useManagementStore.getState().setStreamsIsCreating(false)
      expect(useManagementStore.getState().streamsIsCreating).toBe(false)
    })

    it('sets selected stream name', () => {
      useManagementStore.getState().setStreamsSelectedName('ORDERS')
      expect(useManagementStore.getState().streamsSelectedName).toBe('ORDERS')
      useManagementStore.getState().setStreamsSelectedName(null)
      expect(useManagementStore.getState().streamsSelectedName).toBeNull()
    })
  })

  describe('consumers tab', () => {
    it('sets selected stream', () => {
      useManagementStore.getState().setConsumersSelectedStream('ORDERS')
      expect(useManagementStore.getState().consumersSelectedStream).toBe('ORDERS')
    })

    it('sets creating state', () => {
      useManagementStore.getState().setConsumersIsCreating(true)
      expect(useManagementStore.getState().consumersIsCreating).toBe(true)
    })

    it('sets selected consumer name', () => {
      useManagementStore.getState().setConsumersSelectedName('my-consumer')
      expect(useManagementStore.getState().consumersSelectedName).toBe('my-consumer')
    })
  })

  describe('kv tab', () => {
    it('sets creating state', () => {
      useManagementStore.getState().setKvIsCreating(true)
      expect(useManagementStore.getState().kvIsCreating).toBe(true)
    })

    it('sets selected bucket', () => {
      useManagementStore.getState().setKvSelectedBucket('my-bucket')
      expect(useManagementStore.getState().kvSelectedBucket).toBe('my-bucket')
    })
  })

  describe('objects tab', () => {
    it('sets creating state', () => {
      useManagementStore.getState().setObjectsIsCreating(true)
      expect(useManagementStore.getState().objectsIsCreating).toBe(true)
    })

    it('sets selected bucket', () => {
      useManagementStore.getState().setObjectsSelectedBucket('obj-bucket')
      expect(useManagementStore.getState().objectsSelectedBucket).toBe('obj-bucket')
    })
  })

  describe('reset', () => {
    it('resets all state to defaults', () => {
      // Modify all state
      useManagementStore.getState().setActiveTab('objects')
      useManagementStore.getState().setStreamsIsCreating(true)
      useManagementStore.getState().setStreamsSelectedName('ORDERS')
      useManagementStore.getState().setConsumersSelectedStream('ORDERS')
      useManagementStore.getState().setConsumersIsCreating(true)
      useManagementStore.getState().setKvIsCreating(true)
      useManagementStore.getState().setKvSelectedBucket('bucket')
      useManagementStore.getState().setObjectsIsCreating(true)
      useManagementStore.getState().setObjectsSelectedBucket('obj')

      // Reset
      useManagementStore.getState().reset()

      const state = useManagementStore.getState()
      expect(state.activeTab).toBe('streams')
      expect(state.streamsIsCreating).toBe(false)
      expect(state.streamsSelectedName).toBeNull()
      expect(state.consumersSelectedStream).toBeNull()
      expect(state.consumersIsCreating).toBe(false)
      expect(state.consumersSelectedName).toBeNull()
      expect(state.kvIsCreating).toBe(false)
      expect(state.kvSelectedBucket).toBeNull()
      expect(state.objectsIsCreating).toBe(false)
      expect(state.objectsSelectedBucket).toBeNull()
    })
  })
})
