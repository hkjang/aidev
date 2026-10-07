import sys, json
from playwright.sync_api import sync_playwright

URL = "http://127.0.0.1:5199/__probe__/index.html"
results = []
def check(name, ok, detail):
    results.append((name, ok, detail))
    print(("[PASS] " if ok else "[FAIL] ") + name + " :: " + detail)

with sync_playwright() as p:
    b = p.chromium.launch()
    pg = b.new_page()
    errs = []
    pg.on("pageerror", lambda e: errs.append(str(e)))
    pg.goto(URL, wait_until="load")
    pg.wait_for_selector(".form input[type=number]", timeout=15000)

    num = pg.locator(".form input[type=number]")
    name_in = pg.locator(".form input:not([type=number]):not([type=checkbox])").first
    chk = pg.locator(".form input[type=checkbox]")
    save = pg.locator(".form button")

    def clear_num():
        num.click()
        pg.keyboard.press("Control+a")
        for _ in range(8):
            pg.keyboard.press("Backspace")

    def form_text():
        return pg.locator(".form").inner_text()

    # ── T1: 입력칸 조작 ──────────────────────────────────────────
    clear_num()
    v = num.input_value()
    check("T1a 전부 지우면 빈 칸으로 남는다", v == "", "지운 뒤 input.value=%r" % v)

    pg.keyboard.type("15")
    v = num.input_value()
    check("T1b 지운 뒤 15 를 치면 15 다", v == "15", "타이핑 후 input.value=%r" % v)

    # ── T2: 범위 밖 값이 서버로 나가지 않는다 ────────────────────
    before = form_text()
    clear_num()
    dis_empty = save.is_disabled()
    hint_empty = form_text().replace(before, "")
    check("T2a 빈 칸이면 저장 버튼 비활성", dis_empty, "disabled=%r" % dis_empty)
    check("T2b 빈 칸이면 한국어 안내가 보인다", bool(hint_empty.strip()),
          "추가로 표시된 문구=%r" % hint_empty.strip())

    pg.evaluate("window.__calls.length = 0")
    try:
        save.click(timeout=1500, force=True)
    except Exception as e:
        pass
    calls = pg.evaluate("window.__calls")
    bad = [c for c in calls if not (isinstance(c["args"].get("p_duration"), (int, float))
                                    and 3 <= c["args"]["p_duration"] <= 60)]
    check("T2c 빈 칸 상태로 저장을 눌러도 범위 밖 p_duration 이 안 나간다",
          len(bad) == 0, "전송된 호출=%s" % json.dumps(calls, ensure_ascii=False))

    for bad_v, label in [("0", "0"), ("2", "2(하한 미달)"), ("61", "61(상한 초과)")]:
        clear_num(); pg.keyboard.type(bad_v)
        d = save.is_disabled()
        check("T2d 범위 밖 %s 이면 저장 버튼 비활성" % label, d,
              "input.value=%r disabled=%r" % (num.input_value(), d))

    clear_num(); pg.keyboard.type("15")
    d = save.is_disabled()
    check("T2e 정상값 15 면 저장 버튼 활성", not d, "disabled=%r" % d)
    pg.evaluate("window.__calls.length = 0")
    save.click()
    calls = pg.evaluate("window.__calls")
    ok = len(calls) == 1 and calls[0]["args"].get("p_duration") == 15
    check("T2f 정상값은 숫자 15 로 전송된다", ok, "전송=%s" % json.dumps(calls, ensure_ascii=False))

    # ── T3: 서버 값이 바뀌면 폼이 따라간다 ──────────────────────
    pg.evaluate("window.__setRoom({name:'B방', duration_sec:20, elimination:true})")
    pg.wait_for_timeout(120)
    v = num.input_value()
    check("T3a 서버 duration_sec 변경이 폼에 반영된다", v == "20", "input.value=%r (기대 '20')" % v)
    nv = name_in.input_value()
    check("T3b 서버 name 변경이 폼에 반영된다", nv == "B방", "input.value=%r (기대 'B방')" % nv)
    cv = chk.is_checked()
    check("T3c 서버 elimination 변경이 폼에 반영된다", cv, "checked=%r (기대 True)" % cv)

    # ── T4: 공백 이름 ───────────────────────────────────────────
    name_in.click(); pg.keyboard.press("Control+a"); pg.keyboard.type("   ")
    d = save.is_disabled()
    check("T4 이름이 공백뿐이면 저장 버튼 비활성", d, "disabled=%r" % d)
    name_in.click(); pg.keyboard.press("Control+a"); pg.keyboard.type("B방")

    # ── T5: 4초 폴링이 편집 중 입력을 날리지 않는다 ─────────────
    clear_num(); pg.keyboard.type("45")
    pg.evaluate("window.__setRoom({name:'B방', duration_sec:20, elimination:true})")  # 같은 값, 새 객체
    pg.wait_for_timeout(120)
    v = num.input_value()
    check("T5 값이 같은 폴링 갱신은 편집 중 입력을 보존한다", v == "45",
          "input.value=%r (기대 '45')" % v)

    if errs:
        print("PAGE ERRORS:", errs)
    b.close()

nf = sum(1 for _, ok, _ in results if not ok)
print("\n=== %d passed / %d failed ===" % (len(results) - nf, nf))
sys.exit(1 if nf else 0)
