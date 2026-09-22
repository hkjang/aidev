import React from "react";
import ReactDOM from "react-dom/client";
import "@fontsource-variable/noto-sans-kr";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./styles.css";
import { ensureSecureRandomUUID } from "./pwa/uuid";
import { registerPWA } from "./pwa/register";
ensureSecureRandomUUID();
registerPWA();
ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </React.StrictMode>,
);
