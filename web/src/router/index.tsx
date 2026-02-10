import type { ComponentType } from "react";
import { lazy, Suspense } from "react";
import { createBrowserRouter, Navigate } from "react-router-dom";
import App from "@/App";
import { ErrorBoundary } from "@/components/ErrorBoundary";

import Spinner from "@/components/Spinner";
import { ScheduleProvider } from "@/contexts/ScheduleContext";
import AIChatLayout from "@/layouts/AIChatLayout";
import GeneralLayout from "@/layouts/GeneralLayout";
import MemoLayout from "@/layouts/MemoLayout";
import RootLayout from "@/layouts/RootLayout";
import ScheduleLayout from "@/layouts/ScheduleLayout";
import Home from "@/pages/Home";

const Archived = lazy(() => import("@/pages/Archived"));
const AuthCallback = lazy(() => import("@/pages/AuthCallback"));
const Explore = lazy(() => import("@/pages/Explore"));
const Inboxes = lazy(() => import("@/pages/Inboxes"));
const MemoDetail = lazy(() => import("@/pages/MemoDetail"));
const NotFound = lazy(() => import("@/pages/NotFound"));
const PermissionDenied = lazy(() => import("@/pages/PermissionDenied"));
const Attachments = lazy(() => import("@/pages/Attachments"));
const Setting = lazy(() => import("@/pages/Setting"));
const SignIn = lazy(() => import("@/pages/SignIn"));
const SignUp = lazy(() => import("@/pages/SignUp"));
const UserProfile = lazy(() => import("@/pages/UserProfile"));
const MemoDetailRedirect = lazy(() => import("./MemoDetailRedirect"));
const AIChat = lazy(() => import("@/pages/AIChat"));
const Schedule = lazy(() => import("@/pages/Schedule"));
const Review = lazy(() => import("@/pages/Review"));
const KnowledgeGraph = lazy(() => import("@/pages/KnowledgeGraph"));

import { ROUTES } from "./routes";

// Backward compatibility alias
export const Routes = ROUTES;
export { ROUTES };

// Helper component to reduce Suspense boilerplate for lazy routes
const LazyRoute = ({ component: Component }: { component: ComponentType }) => (
  <Suspense
    fallback={
      <div className="w-full h-64 flex items-center justify-center">
        <Spinner size="lg" />
      </div>
    }
  >
    <Component />
  </Suspense>
);

const router = createBrowserRouter([
  {
    path: "/",
    element: <App />,
    children: [
      {
        path: Routes.AUTH,
        children: [
          { path: "", element: <LazyRoute component={SignIn} /> },
          { path: "signup", element: <LazyRoute component={SignUp} /> },
          { path: "callback", element: <LazyRoute component={AuthCallback} /> },
        ],
      },
      {
        path: Routes.ROOT,
        element: <RootLayout />,
        children: [
          { index: true, element: <Navigate to={Routes.CHAT} replace /> },
          {
            element: <MemoLayout />,
            children: [
              { path: Routes.HOME, element: <Home /> },
              { path: Routes.EXPLORE, element: <LazyRoute component={Explore} /> },
              { path: Routes.ARCHIVED, element: <LazyRoute component={Archived} /> },
              { path: "u/:username", element: <LazyRoute component={UserProfile} /> },
            ],
          },
          // Schedule page with custom layout
          {
            path: Routes.SCHEDULE,
            element: (
              <ScheduleProvider>
                <ScheduleLayout />
              </ScheduleProvider>
            ),
            children: [{ index: true, element: <LazyRoute component={Schedule} /> }],
          },
          // AI Chat page with custom layout
          {
            path: Routes.CHAT,
            element: (
              <ErrorBoundary>
                <AIChatLayout />
              </ErrorBoundary>
            ),
            children: [{ index: true, element: <LazyRoute component={AIChat} /> }],
          },
          {
            element: <GeneralLayout />,
            children: [
              { path: Routes.KNOWLEDGE_GRAPH, element: <LazyRoute component={KnowledgeGraph} /> },
              { path: Routes.ATTACHMENTS, element: <LazyRoute component={Attachments} /> },
              { path: Routes.REVIEW, element: <LazyRoute component={Review} /> },
              { path: Routes.INBOX, element: <LazyRoute component={Inboxes} /> },
              { path: Routes.SETTING, element: <LazyRoute component={Setting} /> },
              { path: "memos/:uid", element: <LazyRoute component={MemoDetail} /> },
              { path: "m/:uid", element: <LazyRoute component={MemoDetailRedirect} /> },
              { path: "403", element: <LazyRoute component={PermissionDenied} /> },
              { path: "404", element: <LazyRoute component={NotFound} /> },
              { path: "*", element: <LazyRoute component={NotFound} /> },
            ],
          },
        ],
      },
    ],
  },
]);

export default router;
