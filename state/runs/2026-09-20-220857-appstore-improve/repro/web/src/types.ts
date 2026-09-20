export type AppStatus =
  "draft" | "pending_review" | "published" | "rejected" | "archived";

export interface VersionInfo {
  version: string;
  commit?: string;
  buildDate?: string;
  goVersion?: string;
  environment?: string;
}

export interface PublicConfig {
  siteName: string;
  /** Landing banner wording; empty or absent means the shipped default. */
  siteDescription?: string;
  heroEyebrow?: string;
  heroTitle?: string;
  heroPrimaryLabel?: string;
  heroSecondaryLabel?: string;
  siteUrl?: string;
  logoUrl?: string;
  faviconUrl?: string;
  publicMode: boolean;
  oidcEnabled: boolean;
  oidcConfigured?: boolean;
  /** SSO may be tried silently (prompt=none) before showing a login screen. */
  oidcAutoLogin?: boolean;
  workflowEnabled: boolean;
  anonymousMcp?: boolean;
  theme?: "light" | "dark" | "system";
}

export interface Category {
  id: string;
  slug: string;
  name: string;
  description?: string;
  icon?: string;
  appCount?: number;
  position?: number;
  active?: boolean;
}

export interface StoreApp {
  id: string;
  slug: string;
  name: string;
  summary: string;
  description?: string;
  iconUrl?: string;
  icon?: string;
  serviceUrl?: string;
  category?: Category;
  categoryId?: string;
  categorySlug?: string;
  categoryName?: string;
  tags?: string[];
  screenshots?: string[];
  language?: string;
  framework?: string;
  supportsMcp?: boolean;
  supportsApi?: boolean;
  featured?: boolean;
  /** Hand-set order on the featured shelf; unranked apps follow by recency. */
  featuredRank?: number | null;
  trending?: boolean;
  trendingScore?: number;
  visibility?: "public" | "private";
  status?: AppStatus;
  version?: string;
  ownerName?: string;
  team?: string;
  createdAt?: string;
  updatedAt?: string;
  /** Latest review of an owned app; /me/apps carries it so owners read why. */
  review?: Review;
  /** SecCheck approval for exactly this content; an edit clears it. */
  securityVerified?: boolean;
  securityVerifiedAt?: string;
}

/** A guide file attached to an app; the bytes stay on the server. */
export interface AppDocument {
  id: string;
  appId: string;
  title: string;
  fileName: string;
  contentType: string;
  size: number;
  checksum?: string;
  uploadedBy?: string;
  uploaderName?: string;
  createdAt: string;
  downloadUrl?: string;
}

export interface User {
  id: string;
  subject?: string;
  username: string;
  displayName?: string;
  email?: string;
  team?: string;
  avatarUrl?: string;
  roles: string[];
  permissions?: string[];
  active?: boolean;
  lastLoginAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface Session {
  authenticated: boolean;
  bootstrapRequired?: boolean;
  bootstrapAvailable?: boolean;
  oidcConfigured?: boolean;
  csrfToken?: string;
  user?: User;
}

export interface PersonalKey {
  id: string;
  name: string;
  type?: "api" | "mcp";
  prefix: string;
  permissions: string[];
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
  revokedAt?: string;
  rotationGraceEndsAt?: string;
  secret?: string;
}

export interface KeyPermissionDefinition {
  key: string;
  name: string;
  description?: string;
  active: boolean;
}

export interface KeyPermissionTemplate {
  id: string;
  name: string;
  description?: string;
  permissions: string[];
}

export interface KeyPolicy {
  maxKeys: number;
  defaultExpiryDays: number;
  rotationGraceDays: number;
  expireUnused: boolean;
  unusedExpiryDays: number;
  forceRotation: boolean;
  forceRotationDays: number;
}

export interface KeyPermissionOptions {
  permissions: KeyPermissionDefinition[];
  templates: KeyPermissionTemplate[];
  policy: KeyPolicy;
}

export interface UserPreferences {
  theme: "light" | "dark" | "system";
  language: string;
  reducedMotion: boolean;
  compactCards: boolean;
}

export interface Review {
  id: string;
  appId: string;
  appName?: string;
  appSlug?: string;
  status: "pending" | "approved" | "rejected" | "cancelled";
  submitterName?: string;
  reviewerName?: string;
  team?: string;
  level?: number;
  reason?: string;
  createdAt?: string;
  updatedAt?: string;
  decidedAt?: string;
}

export interface AuditEntry {
  id: string;
  actor?: string;
  action: string;
  resource?: string;
  ip?: string;
  requestId?: string;
  createdAt: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface ApiProblem {
  code: string;
  message: string;
  requestId?: string;
  details?: unknown;
}

export interface AiStreamEvent {
  event: "message" | "token" | "usage" | "finish" | "error" | "heartbeat";
  data: unknown;
}

export interface AiModelLimit {
  id?: string;
  providerId: string;
  name: string;
  contextWindow: number;
  maxInputTokens: number;
  maxOutputTokens: number;
  enabled: boolean;
}

export interface OidcTestResult {
  ok: boolean;
  issuer: string;
  discoveryUrl: string;
  authorizationEndpoint: string;
  tokenEndpoint: string;
  userInfoEndpoint?: string;
  endSessionEndpoint?: string;
  jwksUri: string;
  scopesSupported?: string[];
  pkceSupported: boolean;
  clientId?: string;
  clientSecretSet?: boolean;
  redirectUrl?: string;
}

export interface BrandingAsset {
  kind: "logo" | "favicon";
  contentType: string;
  size: number;
  updatedAt: string;
  url: string;
}

export interface CspViolation {
  origin: string;
  directive: string;
  page?: string;
  count: number;
  firstSeen: string;
  lastSeen: string;
  allowed: boolean;
}

/** What an owner sees about their app's SecCheck review. */
export interface SecurityCheckView {
  enabled: boolean;
  configured: boolean;
  appId: string;
  appName: string;
  appStatus: AppStatus;
  /** The block to paste into the SecCheck review description. */
  bindingText: string;
  newReviewUrl?: string;
  reviewUrl?: string;
  reviewId?: string;
  reviewNumber?: string;
  status: string;
  finalResult?: string;
  approvedAt?: string;
  checkedAt?: string;
  verified: boolean;
}

/** What the review screen shows: the review plus what it is about. */
export interface ReviewDetail extends Review {
  app?: StoreApp;
  documents?: AppDocument[];
  securityCheck?: SecurityCheckView;
  history?: Review[];
}

export interface SecurityCheckSettings {
  enabled: boolean;
  baseUrl: string;
  apiKeySet: boolean;
  timeoutSeconds: number;
  revision?: number;
  updatedAt?: string;
}

export interface SecurityCheckTestResult {
  ok: boolean;
  account?: string;
  displayName?: string;
  roles?: string[];
}
