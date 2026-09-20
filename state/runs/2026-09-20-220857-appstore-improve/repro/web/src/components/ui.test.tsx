import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it } from "vitest";
import { Button, Dialog, Input, ListInput } from "./ui";

function DialogHarness() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setOpen(true)}>열기</Button>
      <Dialog open={open} title="키 생성" onClose={() => setOpen(false)}>
        <Input aria-label="키 이름" />
        <Button>저장</Button>
      </Dialog>
    </>
  );
}

describe("Dialog accessibility", () => {
  it("labels the dialog, closes with Escape and restores focus", async () => {
    const user = userEvent.setup();
    render(<DialogHarness />);
    const trigger = screen.getByRole("button", { name: "열기" });
    await user.click(trigger);
    expect(screen.getByRole("dialog", { name: "키 생성" })).toBeVisible();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });
});

// The parent re-renders on every keystroke and passes a fresh onClose arrow,
// which is what used to re-run the dialog's focus effect.
function TypingDialogHarness() {
  const [open, setOpen] = useState(true);
  const [value, setValue] = useState("");
  return (
    <Dialog open={open} title="사용자 관리" onClose={() => setOpen(false)}>
      <Input
        aria-label="역할"
        value={value}
        onChange={(event) => setValue(event.target.value)}
      />
    </Dialog>
  );
}

// The dialog focuses its close button from an animation frame, so the steal
// only shows after the frame runs.
const nextFrame = () =>
  new Promise((resolve) => requestAnimationFrame(() => resolve(null)));

describe("Dialog keeps focus while typing", () => {
  it("does not pull focus to the close button on each keystroke", async () => {
    const user = userEvent.setup();
    render(<TypingDialogHarness />);
    // Let the dialog's own open-time focus land before typing.
    await nextFrame();
    const field = screen.getByLabelText("역할");
    await user.click(field);
    await user.keyboard("a");
    await nextFrame();
    expect(field).toHaveFocus();
    await user.keyboard("dmin");
    await nextFrame();
    expect(field).toHaveFocus();
    expect(field).toHaveValue("admin");
  });
});

function ListInputHarness() {
  const [roles, setRoles] = useState<string[]>(["user"]);
  return (
    <>
      <ListInput aria-label="역할 목록" value={roles} onChange={setRoles} />
      <output>{roles.join("|")}</output>
    </>
  );
}

describe("ListInput", () => {
  it("keeps the separator so a second entry can be typed", async () => {
    const user = userEvent.setup();
    render(<ListInputHarness />);
    const field = screen.getByLabelText("역할 목록");
    expect(field).toHaveValue("user");
    await user.click(field);
    await user.keyboard(", admin");
    expect(field).toHaveValue("user, admin");
    expect(screen.getByRole("status")).toHaveTextContent("user|admin");
  });
});
