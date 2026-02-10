export const ROUTES = {
  ROOT: "/",
  HOME: "/memo",
  ATTACHMENTS: "/attachments",
  INBOX: "/inbox",
  ARCHIVED: "/archived",
  SETTING: "/setting",
  EXPLORE: "/explore",
  AUTH: "/auth",
  CHAT: "/chat",
  SCHEDULE: "/schedule",
  REVIEW: "/review",
  KNOWLEDGE_GRAPH: "/knowledge-graph",
} as const;

export type RouteKey = keyof typeof ROUTES;
export type RoutePath = (typeof ROUTES)[RouteKey];
