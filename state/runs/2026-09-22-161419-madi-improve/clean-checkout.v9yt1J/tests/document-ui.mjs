// Follow the same visible controls as a user. No API writes or hidden clicks.
export async function documentTool(page, name) {
  await page
    .getByRole("button", { name: "문서 보기 도구", exact: true })
    .click();
  await page.getByRole("menuitem", { name, exact: true }).click();
}
export async function documentPanel(page, name) {
  const toggle = page.getByRole("button", { name: "문서 패널", exact: true });
  await toggle.waitFor();
  const tab = page.getByRole("tab", { name, exact: true });
  if ((await toggle.getAttribute("aria-expanded")) === "false")
    await toggle.click();
  await tab.waitFor();
  await tab.click();
}
