import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { AppProviders } from "./app/providers";
import { AppRouter } from "./app/router";
import { GlobalErrorBoundary } from "./components/error-boundary";
import { FavoritesProvider } from "./features/apps/favorites";
import "./styles.css";

const root = document.getElementById("root");
if (!root) throw new Error("root element를 찾을 수 없습니다.");

createRoot(root).render(
  <StrictMode>
    <GlobalErrorBoundary>
      <BrowserRouter>
        <AppProviders>
          <FavoritesProvider>
            <AppRouter />
          </FavoritesProvider>
        </AppProviders>
      </BrowserRouter>
    </GlobalErrorBoundary>
  </StrictMode>,
);
