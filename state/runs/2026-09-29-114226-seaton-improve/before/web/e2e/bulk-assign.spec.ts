import { expect, test } from "@playwright/test";
import { readFileSync } from "node:fs";
import { csrfToken, fetchSeats, keepingSeats, login } from "./helpers";

/** 검증용 CSV를 임시 파일 없이 올린다. */
const csv = (body: string) => ({
  name: "assign.csv",
  mimeType: "text/csv",
  buffer: Buffer.from("﻿" + body, "utf8"),
});

test.describe("좌석 일괄 배정", () => {
  test("사번과 좌석 번호 두 열로 배정하고 실패 행은 사유를 보여준다", async ({
    page,
  }) => {
    await login(page);
    await keepingSeats(page, ["이코딩"], async () => {
      const seats = await fetchSeats(page);
      const empty = seats.filter((seat) => !seat.employeeName);
      expect(empty.length, "빈 좌석이 있어야 한다").toBeGreaterThan(1);
      const target = empty[0];
      const taken = seats.find((seat) => seat.employeeName === "김개발")!;

      await page.goto("/admin/employees");
      await page.setInputFiles(
        "input[type=file] >> nth=0",
        csv(
          [
            "사번,좌석번호",
            `E002,${target.seatNo}`,
            `E003,${taken.seatNo}`,
            "E999,없는좌석",
          ].join("\n"),
        ),
      );

      // 한 건은 반영되고, 이미 찬 좌석과 없는 사번은 사유와 함께 남는다.
      await expect(page.getByText(/1건 배정, 2건 확인 필요/)).toBeVisible();
      await expect(page.getByText(/반영되지 않은 2행/)).toBeVisible();
      await expect(
        page.getByText(/이미 다른 직원에게 배정된 좌석입니다/),
      ).toBeVisible();
      await expect(page.getByText(/사번을 찾을 수 없습니다/)).toBeVisible();

      const after = await fetchSeats(page);
      expect(after.find((seat) => seat.id === target.id)?.employeeName).toBe(
        "이코딩",
      );
    });
  });

  test("게시되지 않은 도면의 좌석에는 배정되지 않는다", async ({ page }) => {
    test.setTimeout(120_000);
    await login(page);
    const me = await (await page.request.get("/api/v1/auth/me")).json();
    const headers = { "X-CSRF-Token": me.csrfToken };
    const maps = await (await page.request.get("/api/v1/floor-maps")).json();
    const active = maps.items.find((m: { active?: boolean }) => m.active);

    // 좌석 번호는 도면마다 따로 매겨지므로, 같은 층에 새 버전을 올리면 같은 번호의
    // 좌석이 두 벌 생긴다. 게시된 도면으로 좁히지 않으면 아직 검토 중인 쪽에
    // 배정되어 좌석맵에는 보이지 않는다.
    const draft = await (
      await page.request.post("/api/v1/floor-maps", {
        headers,
        multipart: {
          floorId: active.floorId,
          version: `b${Date.now().toString().slice(-6)}`,
          file: {
            name: "plan.png",
            mimeType: "image/png",
            buffer: readFileSync("e2e/fixtures/plan.png"),
          },
        },
      })
    ).json();
    const job = await (
      await page.request.post(
        `/api/v1/floor-maps/${draft.id}/analyze?engine=cv`,
        { headers },
      )
    ).json();
    await expect
      .poll(
        async () =>
          (
            await (
              await page.request.get(`/api/v1/analysis-jobs/${job.jobId}`)
            ).json()
          ).status,
        { timeout: 90_000 },
      )
      .toBe("completed");

    const seats = await fetchSeats(page);
    const target = seats.find((seat) => !seat.employeeName)!;
    await keepingSeats(page, ["최프론트"], async () => {
      const response = await page.request.post(
        "/api/v1/seat-assignments/bulk",
        {
          headers,
          multipart: {
            file: csv(`사번,좌석번호\nE004,${target.seatNo}`),
          },
        },
      );
      const result = await response.json();
      expect(result.success).toBe(1);

      const after = await (
        await page.request.get(`/api/v1/seats?floorMapId=${active.id}`)
      ).json();
      const assigned = after.items.find(
        (seat: { id: string }) => seat.id === target.id,
      );
      expect(
        assigned.employeeName,
        "게시된 도면의 좌석이 배정되어야 한다",
      ).toBe("최프론트");
    });
    await page.request.delete(`/api/v1/floor-maps/${draft.id}`, { headers });
  });
});
