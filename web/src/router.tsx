import { createBrowserRouter, Navigate } from 'react-router-dom'
import { lazy } from 'react'
import RootLayout from './components/common/RootLayout'
import ConnectedLayout from './components/common/ConnectedLayout'
import NotFoundPage from './components/common/NotFoundPage'
import { LazyRoute } from './components/common/LazyRoute'
import StreamView from './components/streams/StreamView'
import ConnectionSelector from './components/connections/ConnectionSelector'

// Stream tab components (eagerly loaded as they're commonly used)
import MessagesTab from './components/streams/MessagesTab'
import PublishTab from './components/streams/PublishTab'

// Lazy loaded components for better initial bundle size
const CreateStreamPage = lazy(() => import('./components/streams/CreateStreamPage'))
const StreamConfigTab = lazy(() => import('./components/streams/StreamConfigTab'))
const StreamConsumersTab = lazy(() => import('./components/streams/StreamConsumersTab'))
const KVStorePage = lazy(() => import('./components/kv/KVStorePage'))
const CreateKVPage = lazy(() => import('./components/kv/CreateKVPage'))
const KVOverviewPage = lazy(() => import('./components/kv/KVOverviewPage'))
const ObjectsTab = lazy(() => import('./components/management/objects/ObjectsTab'))
const ObjectsOverviewPage = lazy(() => import('./components/objects/ObjectsOverviewPage'))
// Settings pages are eager-loaded: lazy-load + Suspense caused a
// "Loading..." flash on every tab switch.
import SettingsLayout from './components/settings/page/SettingsLayout'
import ConnectionsPage from './components/settings/page/ConnectionsPage'
import ConnectionEditPage from './components/settings/page/ConnectionEditPage'
import ProtoPage from './components/settings/page/ProtoPage'
import ProtoSourceEditPage from './components/settings/page/ProtoSourceEditPage'
import MappingsPage from './components/settings/page/MappingsPage'
import PreferencesPage from './components/settings/page/PreferencesPage'
import WorkspacePage from './components/settings/page/WorkspacePage'
import TemplatesPage from './components/settings/page/TemplatesPage'

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [
      // Root - connection selector
      { index: true, element: <ConnectionSelector /> },

      // Connected state routes (pathless layout — guards via localStorage)
      {
        element: <ConnectedLayout />,
        children: [
          // Streams
          {
            path: 'streams',
            children: [
              { index: true, element: (
                <div className="flex-1 flex items-center justify-center text-content-muted">
                  <div className="text-center">
                    <svg className="w-12 h-12 mx-auto mb-3 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                    <p className="text-sm">Select a stream from the sidebar</p>
                  </div>
                </div>
              ) },
              // Create new stream - must be before :streamName
              { path: 'new', element: <LazyRoute><CreateStreamPage /></LazyRoute> },
              {
                path: ':streamName',
                element: <StreamView />,
                children: [
                  { index: true, element: <Navigate to="messages" replace /> },
                  { path: 'messages', element: <MessagesTab /> },
                  { path: 'config', element: <LazyRoute><StreamConfigTab /></LazyRoute> },
                  { path: 'consumers', element: <LazyRoute><StreamConsumersTab /></LazyRoute> },
                  { path: 'publish', element: <PublishTab /> },
                  // Redirect old 'info' route to 'config'
                  { path: 'info', element: <Navigate to="../config" replace /> },
                ]
              }
            ]
          },

          // KV Stores
          {
            path: 'kv',
            children: [
              { index: true, element: <LazyRoute><KVOverviewPage /></LazyRoute> },
              // Create new KV store - must be before :bucketName
              { path: 'new', element: <LazyRoute><CreateKVPage /></LazyRoute> },
              // KV store view
              { path: ':bucketName', element: <LazyRoute><KVStorePage /></LazyRoute> },
            ]
          },

          // Object Store
          {
            path: 'objects',
            children: [
              { index: true, element: <LazyRoute><ObjectsOverviewPage /></LazyRoute> },
              // Create new bucket - must be before :bucketName
              { path: 'new', element: <LazyRoute><ObjectsTab createMode /></LazyRoute> },
              // Object store view
              { path: ':bucketName', element: <LazyRoute><ObjectsTab /></LazyRoute> },
            ]
          },

          // Settings (full-page version of the modal)
          {
            path: 'settings',
            element: <LazyRoute><SettingsLayout /></LazyRoute>,
            children: [
              { index: true, element: <Navigate to="connections" replace /> },
              { path: 'connections', element: <LazyRoute><ConnectionsPage /></LazyRoute> },
              { path: 'connections/new', element: <LazyRoute><ConnectionEditPage mode="create" /></LazyRoute> },
              { path: 'connections/:id/edit', element: <LazyRoute><ConnectionEditPage mode="edit" /></LazyRoute> },
              { path: 'proto', element: <LazyRoute><ProtoPage /></LazyRoute> },
              { path: 'proto/new', element: <LazyRoute><ProtoSourceEditPage mode="create" /></LazyRoute> },
              { path: 'proto/:id/edit', element: <LazyRoute><ProtoSourceEditPage mode="edit" /></LazyRoute> },
              { path: 'mappings', element: <LazyRoute><MappingsPage /></LazyRoute> },
              { path: 'templates', element: <LazyRoute><TemplatesPage /></LazyRoute> },
              { path: 'preferences', element: <LazyRoute><PreferencesPage /></LazyRoute> },
              { path: 'workspace', element: <LazyRoute><WorkspacePage /></LazyRoute> },
            ],
          },

          // Legacy routes - redirect to new paths
          { path: 'storage', element: <Navigate to="/objects" replace /> },
          { path: 'storage/kv', element: <Navigate to="/kv" replace /> },
          { path: 'storage/objects', element: <Navigate to="/objects" replace /> },
          { path: 'management', element: <Navigate to="/objects" replace /> },
          { path: 'management/streams', element: <Navigate to="/streams" replace /> },
          { path: 'management/consumers', element: <Navigate to="/streams" replace /> },
          { path: 'management/kv', element: <Navigate to="/kv" replace /> },
          { path: 'management/objects', element: <Navigate to="/objects" replace /> },
          { path: 'buckets', element: <Navigate to="/objects" replace /> },
        ]
      },

      // Legacy /c/:connectionId routes - redirect to new flat paths
      { path: 'c/:connectionId/streams/*', element: <Navigate to="/streams" replace /> },
      { path: 'c/:connectionId/kv/*', element: <Navigate to="/kv" replace /> },
      { path: 'c/:connectionId/objects/*', element: <Navigate to="/objects" replace /> },
      { path: 'c/:connectionId', element: <Navigate to="/streams" replace /> },

      // 404 for unknown routes at root level
      { path: '*', element: <NotFoundPage fullScreen /> },
    ]
  }
])
