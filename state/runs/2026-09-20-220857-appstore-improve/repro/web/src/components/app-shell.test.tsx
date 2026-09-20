import { render, screen } from "@testing-library/react";
import { MemoryRouter, NavLink } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { navItemActive } from "./app-shell";

describe("navItemActive", () => {
  it("keeps Apps and MCP Apps mutually exclusive", () => {
    expect(navItemActive("/apps", "/apps", "")).toBe(true);
    expect(navItemActive("/apps?mcp=true", "/apps", "")).toBe(false);
    expect(navItemActive("/apps", "/apps", "?mcp=true")).toBe(false);
    expect(navItemActive("/apps?mcp=true", "/apps", "?mcp=true")).toBe(true);
  });

  it("ignores unrelated query parameters on the plain entry", () => {
    expect(navItemActive("/apps", "/apps", "?q=agent&sort=trending")).toBe(
      true,
    );
  });

  it("matches nested paths unless the entry is exact", () => {
    expect(navItemActive("/admin/apps", "/admin/apps/abc", "")).toBe(true);
    expect(navItemActive("/admin", "/admin/apps", "", true)).toBe(false);
  });
});

describe("nav link rendering", () => {
  // NavLink appends its own pathname-only "active" class when className is a
  // string, which lit up Apps and MCP Apps together. The callback form is what
  // keeps our query-aware result authoritative.
  it("takes the active class only from the callback className", () => {
    render(
      <MemoryRouter initialEntries={["/apps?mcp=true"]}>
        <NavLink to="/apps" className={() => "nav-link"}>
          Apps
        </NavLink>
        <NavLink to="/apps" className="string-link">
          String
        </NavLink>
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "Apps" })).toHaveAttribute(
      "class",
      "nav-link",
    );
    expect(screen.getByRole("link", { name: "String" })).toHaveAttribute(
      "class",
      "string-link active",
    );
  });
});
