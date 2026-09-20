import { Navigate, Route, Routes } from "react-router-dom";
import { PublicLayout, RequireAuth } from "../components/route-guards";
import { ForbiddenState, NotFoundState } from "../components/ui";
import { AppFormPage } from "../pages/app-form-page";
import { AdminSecurityCheckPage } from "../features/security-check/admin-page";
import { AppSecurityCheckPage } from "../features/security-check/owner-page";
import { LoginPage } from "../pages/auth-pages";
import {
  AdminAiPage,
  AdminAnalyticsPage,
  AdminApiKeysPage,
  AdminApiPage,
  AdminAppDetailPage,
  AdminAppsPage,
  AdminAuditPage,
  AdminAuthenticationPage,
  AdminCategoriesPage,
  AdminDashboardPage,
  AdminMcpPage,
  AdminReviewsPage,
  AdminRolesPage,
  AdminSecurityPage,
  AdminSystemSettingsPage,
  AdminUsersPage,
  AdminWorkflowPage,
} from "../pages/admin-pages";
import {
  ActivityPage,
  MyAppsPage,
  MyDashboardPage,
  MyKeysPage,
  PersonalSettingsPage,
  ProfilePage,
} from "../pages/personal-pages";
import {
  AppDetailPage,
  AppsPage,
  CategoriesPage,
  CategoryPage,
  FavoritesPage,
  HomePage,
  SearchCompatibilityPage,
} from "../pages/public-pages";
import { ReviewDetailPage, ReviewQueuePage } from "../pages/review-pages";

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/admin/bootstrap"
        element={
          <Navigate to="/login?returnTo=%2Fadmin%2Fauthentication" replace />
        }
      />
      <Route element={<PublicLayout />}>
        <Route index element={<HomePage />} />
        <Route path="today" element={<HomePage />} />
        <Route path="apps" element={<AppsPage />} />
        <Route path="apps/:slug" element={<AppDetailPage />} />
        <Route path="categories" element={<CategoriesPage />} />
        <Route path="categories/:category" element={<CategoryPage />} />
        <Route path="search" element={<SearchCompatibilityPage />} />
        <Route path="favorites" element={<FavoritesPage />} />
        <Route
          path="403"
          element={
            <div className="page">
              <ForbiddenState />
            </div>
          }
        />
        <Route
          path="*"
          element={
            <div className="page">
              <NotFoundState />
            </div>
          }
        />
      </Route>

      <Route element={<RequireAuth />}>
        <Route path="submit" element={<AppFormPage />} />
        <Route path="my" element={<MyDashboardPage />} />
        <Route path="my/apps" element={<MyAppsPage />} />
        <Route path="my/apps/:id/edit" element={<AppFormPage edit />} />
        <Route path="my/apps/:id/security" element={<AppSecurityCheckPage />} />
        <Route path="my/keys" element={<MyKeysPage />} />
        <Route path="my/profile" element={<ProfilePage />} />
        <Route path="my/activity" element={<ActivityPage />} />
        <Route path="my/settings" element={<PersonalSettingsPage />} />
        <Route
          path="my/favorites"
          element={<Navigate to="/favorites" replace />}
        />
      </Route>

      <Route
        element={
          <RequireAuth
            roles={["reviewer", "team_leader", "admin", "super_admin"]}
          />
        }
      >
        <Route path="review" element={<ReviewQueuePage />} />
        <Route path="review/:id" element={<ReviewDetailPage />} />
      </Route>

      <Route element={<RequireAuth roles={["admin", "super_admin"]} admin />}>
        <Route path="admin" element={<AdminDashboardPage />} />
        <Route path="admin/apps" element={<AdminAppsPage />} />
        <Route path="admin/apps/new" element={<AdminAppDetailPage create />} />
        <Route path="admin/apps/:id" element={<AdminAppDetailPage />} />
        <Route path="admin/categories" element={<AdminCategoriesPage />} />
        <Route path="admin/users" element={<AdminUsersPage />} />
        <Route path="admin/roles" element={<AdminRolesPage />} />
        <Route path="admin/reviews" element={<AdminReviewsPage />} />
        <Route path="admin/workflow" element={<AdminWorkflowPage />} />
        <Route path="admin/ai" element={<AdminAiPage />} />
        <Route path="admin/api" element={<AdminApiPage />} />
        <Route path="admin/mcp" element={<AdminMcpPage />} />
        <Route path="admin/api-keys" element={<AdminApiKeysPage />} />
        <Route
          path="admin/authentication"
          element={<AdminAuthenticationPage />}
        />
        <Route path="admin/security" element={<AdminSecurityPage />} />
        <Route
          path="admin/security-check"
          element={<AdminSecurityCheckPage />}
        />
        <Route path="admin/analytics" element={<AdminAnalyticsPage />} />
        <Route path="admin/audit" element={<AdminAuditPage />} />
        <Route path="admin/settings" element={<AdminSystemSettingsPage />} />
      </Route>
    </Routes>
  );
}
