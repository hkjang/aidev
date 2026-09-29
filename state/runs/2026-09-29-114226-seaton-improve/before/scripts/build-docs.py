#!/usr/bin/env python3
"""docs/*.md 에서 배포용 HTML과 PDF를 생성한다.

이전에는 HTML과 PDF를 손으로 따로 작성해 Markdown 본문과 내용이 어긋났다.
이 스크립트가 단일 원본(Markdown)에서 두 산출물을 만들어 그 문제를 없앤다.

사용법:
    python3 scripts/build-docs.py            # docs 전체 재생성
    python3 scripts/build-docs.py ADMIN_GUIDE USER_GUIDE

PDF는 reportlab과 docs/fonts/NanumGothic.ttf 를 사용한다(한글 임베드).
    pip install reportlab
"""

from __future__ import annotations

import html
import re
import sys
from dataclasses import dataclass, field
from pathlib import Path

DOCS = Path(__file__).resolve().parent.parent / "docs"
FONT_DIR = DOCS / "fonts"

# PDF 를 공용 도구(aidev/tools/guide/md2pdf.mjs)로 굽는 문서. 이 스크립트는 HTML 만 만든다.
# 가이드 PDF 서식을 다른 프로젝트와 맞추기 위해서다. 여기서도 PDF 를 만들면 두 서식이 번갈아 덮어쓴다.
SHARED_PDF = {"ADMIN_GUIDE", "USER_GUIDE"}

# 생성 대상과 HTML <title>. Markdown 파일명을 기준으로 한다.
TARGETS = {
    "ADMIN_GUIDE": "SeatOn 엔터프라이즈 관리자 가이드 (Admin Guide)",
    "USER_GUIDE": "SeatOn 사용자 가이드 (User Guide)",
    "EXECUTIVE_REPORT": "SeatOn 경영 보고서 (Executive Report)",
    "ROADMAP_PLAN": "SeatOn 로드맵 계획서 (Roadmap Plan)",
    "USER_GROUPS_ANALYSIS": "SeatOn 타겟 사용자군 분석 (User Groups Analysis)",
}

# ─────────────────────────────── Markdown 파싱 ───────────────────────────────


@dataclass
class Block:
    kind: str  # heading | paragraph | list | table | code | quote | rule | image
    level: int = 0
    text: str = ""
    items: list[str] = field(default_factory=list)
    ordered: bool = False
    start: int = 1
    rows: list[list[str]] = field(default_factory=list)
    src: str = ""


@dataclass
class Document:
    title: str
    meta: list[tuple[str, str]]
    blocks: list[Block]


META_LINE = re.compile(r"^-\s+\*\*(.+?)\*\*\s*:\s*(.*?)\s*$")
# 화면 캡처 한 줄. alt 텍스트가 그림 설명이 된다.
IMAGE_LINE = re.compile(r"^!\[([^\]]*)\]\(([^)\s]+)\)$")


def parse_markdown(text: str) -> Document:
    lines = text.replace("\r\n", "\n").split("\n")
    title = ""
    meta: list[tuple[str, str]] = []
    blocks: list[Block] = []
    index = 0

    # 문서 제목(첫 h1)과 그 뒤에 이어지는 메타 불릿을 헤더로 분리한다.
    while index < len(lines):
        line = lines[index]
        if not title and line.startswith("# "):
            title = line[2:].strip()
            index += 1
            continue
        if title and (matched := META_LINE.match(line.rstrip())):
            meta.append((matched.group(1), matched.group(2)))
            index += 1
            continue
        if title and line.strip() == "":
            index += 1
            if meta:
                break
            continue
        break

    paragraph: list[str] = []
    items: list[str] = []
    ordered = False
    start = 1
    table: list[list[str]] = []

    def flush() -> None:
        nonlocal paragraph, items, table, start
        if paragraph:
            blocks.append(Block("paragraph", text=" ".join(paragraph).strip()))
            paragraph = []
        if items:
            blocks.append(Block("list", items=items, ordered=ordered, start=start))
            start += len(items)
            items = []
        if table:
            blocks.append(Block("table", rows=table))
            table = []

    while index < len(lines):
        raw = lines[index]
        line = raw.rstrip()
        stripped = line.strip()

        if stripped.startswith("```"):
            flush()
            index += 1
            body: list[str] = []
            while index < len(lines) and not lines[index].strip().startswith("```"):
                body.append(lines[index])
                index += 1
            index += 1
            blocks.append(Block("code", text="\n".join(body)))
            continue

        if re.match(r"^#{1,4}\s", stripped):
            flush()
            level = len(stripped) - len(stripped.lstrip("#"))
            blocks.append(Block("heading", level=level, text=stripped[level:].strip()))
            index += 1
            continue

        if re.match(r"^(-{3,}|\*{3,}|_{3,})$", stripped):
            flush()
            blocks.append(Block("rule"))
            index += 1
            continue

        if stripped.startswith("|") and stripped.endswith("|"):
            cells = [c.strip() for c in stripped.strip("|").split("|")]
            # 헤더 구분선(---)은 표의 일부이지만 내용이 아니다.
            if not all(re.fullmatch(r":?-{2,}:?", c) for c in cells):
                if paragraph or items:
                    flush()
                table.append(cells)
            index += 1
            continue

        image = IMAGE_LINE.match(stripped)
        if image:
            flush()
            blocks.append(Block("image", text=image.group(1), src=image.group(2)))
            index += 1
            continue

        if stripped.startswith("> "):
            flush()
            quote = [stripped[2:].strip()]
            index += 1
            while index < len(lines) and lines[index].strip().startswith(">"):
                quote.append(lines[index].strip().lstrip(">").strip())
                index += 1
            blocks.append(Block("quote", text=" ".join(q for q in quote if q)))
            continue

        bullet = re.match(r"^([-*+])\s+(.*)$", stripped)
        number = re.match(r"^(\d+)\.\s+(.*)$", stripped)
        if bullet or number:
            if table:
                flush()
            if paragraph:
                blocks.append(Block("paragraph", text=" ".join(paragraph).strip()))
                paragraph = []
            wanted_ordered = number is not None
            if items and wanted_ordered != ordered:
                blocks.append(Block("list", items=items, ordered=ordered, start=start))
                items = []
            # 번호 목록 사이에 끼운 캡처는 목록을 끊지만 번호는 이어진다.
            # 그 밖의 경우는 새 목록이므로 1부터 센다.
            if not items and (not blocks or blocks[-1].kind != "image"):
                start = 1
            ordered = wanted_ordered
            items.append((number or bullet).group(2).strip())
            index += 1
            continue

        # 목록 항목의 이어지는 줄(들여쓰기)은 직전 항목에 붙인다.
        if items and raw.startswith(("  ", "\t")) and stripped:
            items[-1] += " " + stripped
            index += 1
            continue

        if stripped == "":
            flush()
            index += 1
            continue

        if table:
            flush()
        paragraph.append(stripped)
        index += 1

    flush()
    return Document(title=title, meta=meta, blocks=blocks)


# ──────────────────────────────── HTML 생성 ────────────────────────────────

INLINE_CODE = re.compile(r"`([^`]+)`")
BOLD = re.compile(r"\*\*(.+?)\*\*")
LINK = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")


def inline_html(text: str) -> str:
    out = html.escape(text, quote=False)
    out = INLINE_CODE.sub(lambda m: f"<code>{m.group(1)}</code>", out)
    out = BOLD.sub(lambda m: f"<strong>{m.group(1)}</strong>", out)
    out = LINK.sub(lambda m: f'<a href="{m.group(2)}">{m.group(1)}</a>', out)
    return out


HTML_TEMPLATE = """<!DOCTYPE html>
<html lang="ko">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{title}</title>
    <meta name="description" content="{description}">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Plus+Jakarta+Sans:wght@700;800&family=JetBrains+Mono:wght@400;600&display=swap" rel="stylesheet">
    <style>
        :root {{
            --primary: #10b981;
            --primary-dark: #047857;
            --text-dark: #0f172a;
            --text-muted: #475569;
            --bg-light: #f8fafc;
            --border-color: #e2e8f0;
        }}
        * {{ box-sizing: border-box; margin: 0; padding: 0; }}
        body {{ font-family: 'Inter', -apple-system, sans-serif; color: var(--text-dark); background: var(--bg-light); line-height: 1.65; padding: 48px 24px; }}
        .doc-card {{ max-width: 1040px; margin: 0 auto; background: #ffffff; border: 1px solid var(--border-color); border-radius: 20px; padding: 56px; box-shadow: 0 12px 30px rgba(0,0,0,0.06); }}
        .header-meta {{ border-bottom: 3px solid var(--primary); padding-bottom: 28px; margin-bottom: 36px; }}
        .header-meta h1 {{ font-family: 'Plus Jakarta Sans', sans-serif; font-size: 2.2rem; color: #0f172a; margin-bottom: 16px; letter-spacing: -0.02em; }}
        .meta-grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; font-size: 0.9rem; color: var(--text-muted); }}
        .meta-item {{ background: #f1f5f9; padding: 8px 14px; border-radius: 8px; border: 1px solid #e2e8f0; }}
        h2 {{ font-family: 'Plus Jakarta Sans', sans-serif; font-size: 1.45rem; color: var(--primary-dark); margin: 40px 0 20px; border-bottom: 2px solid #f1f5f9; padding-bottom: 10px; }}
        h3 {{ font-family: 'Plus Jakarta Sans', sans-serif; font-size: 1.12rem; color: #0f172a; margin: 28px 0 14px; }}
        h4 {{ font-size: 1rem; color: var(--text-muted); margin: 22px 0 10px; }}
        p {{ margin-bottom: 18px; font-size: 1rem; color: #334155; }}
        ul, ol {{ margin-bottom: 20px; padding-left: 24px; }}
        li {{ margin-bottom: 8px; }}
        pre {{ background: #0f172a; color: #f8fafc; padding: 24px; border-radius: 12px; font-family: 'JetBrains Mono', monospace; font-size: 0.88rem; overflow-x: auto; margin: 20px 0; }}
        code {{ font-family: 'JetBrains Mono', monospace; background: #e2e8f0; color: #0f172a; padding: 2px 6px; border-radius: 4px; font-size: 0.88rem; }}
        pre code {{ background: none; color: inherit; padding: 0; }}
        table {{ width: 100%; border-collapse: collapse; margin: 20px 0; font-size: 0.92rem; }}
        th, td {{ border: 1px solid var(--border-color); padding: 10px 12px; text-align: left; vertical-align: top; }}
        th {{ background: #f1f5f9; font-weight: 700; color: #0f172a; }}
        tbody tr:nth-child(even) {{ background: #fafafa; }}
        blockquote {{ border-left: 4px solid var(--primary); background: #ecfdf5; padding: 14px 18px; border-radius: 0 10px 10px 0; margin: 20px 0; color: #065f46; }}
        hr {{ border: none; border-top: 1px solid var(--border-color); margin: 32px 0; }}
        figure {{ margin: 20px 0 24px; }}
        figure img {{ width: 100%; border: 1px solid var(--border-color); border-radius: 10px; }}
        figcaption {{ margin-top: 8px; font-size: 0.88rem; color: var(--text-muted); }}
        a {{ color: var(--primary-dark); }}
        .doc-nav {{ margin-bottom: 24px; font-size: 0.9rem; }}
        .print-btn {{ display: inline-block; background: var(--primary); color: #ffffff; padding: 12px 24px; border-radius: 10px; text-decoration: none; font-weight: 700; font-size: 0.92rem; float: right; }}
        .print-btn:hover {{ background: var(--primary-dark); }}
        .generated {{ margin-top: 40px; padding-top: 20px; border-top: 1px solid var(--border-color); font-size: 0.82rem; color: var(--text-muted); }}
        @media print {{ .print-btn, .doc-nav {{ display: none; }} body {{ background: #ffffff; padding: 0; }} .doc-card {{ border: none; box-shadow: none; padding: 0; }} }}
    </style>
</head>
<body>
    <div class="doc-card">
        <a href="javascript:window.print()" class="print-btn">🖨️ PDF / 인쇄하기</a>
        <div class="header-meta">
            <h1>{heading}</h1>
{meta_html}        </div>
{body}
        <div class="generated">이 문서는 <code>{source}</code> 에서 <code>scripts/build-docs.py</code> 로 생성되었습니다. 내용 수정은 Markdown 원본에서 하십시오.</div>
    </div>
</body>
</html>
"""


def render_html(doc: Document, name: str, title: str) -> str:
    parts: list[str] = []
    for block in doc.blocks:
        if block.kind == "heading":
            level = min(max(block.level, 2), 4)
            parts.append(f"        <h{level}>{inline_html(block.text)}</h{level}>")
        elif block.kind == "paragraph":
            parts.append(f"        <p>{inline_html(block.text)}</p>")
        elif block.kind == "list":
            tag = "ol" if block.ordered else "ul"
            attrs = f' start="{block.start}"' if block.ordered and block.start > 1 else ""
            entries = "".join(
                f"\n            <li>{inline_html(item)}</li>" for item in block.items
            )
            parts.append(f"        <{tag}{attrs}>{entries}\n        </{tag}>")
        elif block.kind == "image":
            alt = html.escape(block.text, quote=True)
            parts.append(
                f'        <figure><img src="{html.escape(block.src, quote=True)}" alt="{alt}" loading="lazy">'
                f"<figcaption>{inline_html(block.text)}</figcaption></figure>"
            )
        elif block.kind == "code":
            parts.append(
                "        <pre><code>"
                + html.escape(block.text, quote=False)
                + "</code></pre>"
            )
        elif block.kind == "quote":
            parts.append(f"        <blockquote>{inline_html(block.text)}</blockquote>")
        elif block.kind == "rule":
            parts.append("        <hr>")
        elif block.kind == "table" and block.rows:
            header, *body = block.rows
            head = "".join(f"<th>{inline_html(c)}</th>" for c in header)
            rows = "".join(
                "\n                <tr>"
                + "".join(f"<td>{inline_html(c)}</td>" for c in row)
                + "</tr>"
                for row in body
            )
            parts.append(
                "        <table>\n            <thead><tr>"
                + head
                + "</tr></thead>\n            <tbody>"
                + rows
                + "\n            </tbody>\n        </table>"
            )
    meta_html = ""
    if doc.meta:
        cells = "".join(
            f'\n                <div class="meta-item"><strong>{inline_html(k)}:</strong> {inline_html(v)}</div>'
            for k, v in doc.meta
        )
        meta_html = f'            <div class="meta-grid">{cells}\n            </div>\n'
    description = next(
        (b.text for b in doc.blocks if b.kind == "paragraph" and len(b.text) > 40),
        doc.title,
    )
    description = re.sub(r"[*`\[\]()]", "", description)[:200]
    return HTML_TEMPLATE.format(
        title=title,
        heading=inline_html(doc.title or title),
        description=html.escape(description, quote=True),
        meta_html=meta_html,
        body="\n".join(parts),
        source=f"docs/{name}.md",
    )


# ───────────────────────────────── PDF 생성 ─────────────────────────────────


def render_pdf(doc: Document, path: Path) -> None:
    from reportlab.lib import colors
    from reportlab.lib.enums import TA_LEFT
    from reportlab.lib.pagesizes import A4
    from reportlab.lib.styles import ParagraphStyle
    from reportlab.lib.units import mm
    from reportlab.pdfbase import pdfmetrics
    from reportlab.pdfbase.ttfonts import TTFont
    from reportlab.platypus import (
        HRFlowable,
        KeepTogether,
        ListFlowable,
        ListItem,
        PageBreak,
        Paragraph,
        SimpleDocTemplate,
        Spacer,
        Table,
        TableStyle,
    )

    regular, bold = FONT_DIR / "NanumGothic.ttf", FONT_DIR / "NanumGothic-Bold.ttf"
    if not regular.exists() or not bold.exists():
        raise SystemExit(f"한글 폰트를 찾을 수 없습니다: {FONT_DIR}")
    pdfmetrics.registerFont(TTFont("NanumGothic", str(regular)))
    pdfmetrics.registerFont(TTFont("NanumGothicBold", str(bold)))
    pdfmetrics.registerFontFamily(
        "NanumGothic", normal="NanumGothic", bold="NanumGothicBold"
    )

    ink, muted, green = colors.HexColor("#0f172a"), colors.HexColor("#475569"), colors.HexColor("#047857")
    body = ParagraphStyle(
        "body", fontName="NanumGothic", fontSize=9.5, leading=15,
        textColor=colors.HexColor("#334155"), alignment=TA_LEFT, spaceAfter=7,
    )
    h1 = ParagraphStyle("h1", parent=body, fontName="NanumGothicBold", fontSize=19,
                        leading=25, textColor=ink, spaceAfter=10)
    h2 = ParagraphStyle("h2", parent=body, fontName="NanumGothicBold", fontSize=13.5,
                        leading=19, textColor=green, spaceBefore=16, spaceAfter=7)
    h3 = ParagraphStyle("h3", parent=body, fontName="NanumGothicBold", fontSize=11,
                        leading=16, textColor=ink, spaceBefore=11, spaceAfter=5)
    h4 = ParagraphStyle("h4", parent=body, fontName="NanumGothicBold", fontSize=10,
                        leading=15, textColor=muted, spaceBefore=9, spaceAfter=4)
    metaStyle = ParagraphStyle("meta", parent=body, fontSize=8.5, leading=12.5, textColor=muted)
    codeStyle = ParagraphStyle("code", parent=body, fontName="NanumGothic", fontSize=8.2,
                               leading=12, textColor=colors.HexColor("#e2e8f0"))
    quoteStyle = ParagraphStyle("quote", parent=body, fontSize=9.2, leading=14,
                                textColor=colors.HexColor("#065f46"), leftIndent=8)
    captionStyle = ParagraphStyle("caption", parent=body, fontSize=8.4, leading=12,
                                  textColor=colors.HexColor("#475569"), spaceBefore=4)
    cell = ParagraphStyle("cell", parent=body, fontSize=8.4, leading=12, spaceAfter=0)
    cellHead = ParagraphStyle("cellHead", parent=cell, fontName="NanumGothicBold", textColor=ink)

    def inline_pdf(text: str) -> str:
        out = html.escape(text, quote=False)
        out = INLINE_CODE.sub(
            lambda m: f'<font face="NanumGothicBold" color="#0f172a">{m.group(1)}</font>', out)
        out = BOLD.sub(lambda m: f"<b>{m.group(1)}</b>", out)
        out = LINK.sub(lambda m: f'<font color="#047857">{m.group(1)}</font>', out)
        return out

    frame_width = A4[0] - 32 * mm
    story: list = []
    if doc.title:
        story.append(Paragraph(inline_pdf(doc.title), h1))
    if doc.meta:
        rows = [[Paragraph(f"<b>{inline_pdf(k)}</b>", metaStyle), Paragraph(inline_pdf(v), metaStyle)]
                for k, v in doc.meta]
        meta_table = Table(rows, colWidths=[frame_width * 0.24, frame_width * 0.76])
        meta_table.setStyle(TableStyle([
            ("BACKGROUND", (0, 0), (-1, -1), colors.HexColor("#f1f5f9")),
            ("BOX", (0, 0), (-1, -1), 0.5, colors.HexColor("#e2e8f0")),
            ("INNERGRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#e2e8f0")),
            ("VALIGN", (0, 0), (-1, -1), "TOP"),
            ("LEFTPADDING", (0, 0), (-1, -1), 6), ("RIGHTPADDING", (0, 0), (-1, -1), 6),
            ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
        ]))
        story.append(meta_table)
    story.append(Spacer(1, 6))
    story.append(HRFlowable(width="100%", thickness=1.6, color=colors.HexColor("#10b981")))
    story.append(Spacer(1, 8))

    for block in doc.blocks:
        if block.kind == "heading":
            style = {2: h2, 3: h3}.get(block.level, h4) if block.level > 1 else h2
            story.append(Paragraph(inline_pdf(block.text), style))
        elif block.kind == "paragraph":
            story.append(Paragraph(inline_pdf(block.text), body))
        elif block.kind == "list":
            entries = [ListItem(Paragraph(inline_pdf(i), body), leftIndent=12)
                       for i in block.items]
            story.append(ListFlowable(
                entries, bulletType="1" if block.ordered else "bullet",
                bulletFontName="NanumGothic", bulletFontSize=8.5,
                leftIndent=14, spaceAfter=6))
        elif block.kind == "code":
            lines = [Paragraph(inline_pdf(l) or "&nbsp;", codeStyle)
                     for l in block.text.split("\n")]
            table = Table([[lines]], colWidths=[frame_width])
            table.setStyle(TableStyle([
                ("BACKGROUND", (0, 0), (-1, -1), colors.HexColor("#0f172a")),
                ("LEFTPADDING", (0, 0), (-1, -1), 10), ("RIGHTPADDING", (0, 0), (-1, -1), 10),
                ("TOPPADDING", (0, 0), (-1, -1), 8), ("BOTTOMPADDING", (0, 0), (-1, -1), 8),
            ]))
            story.append(table)
            story.append(Spacer(1, 8))
        elif block.kind == "quote":
            table = Table([[Paragraph(inline_pdf(block.text), quoteStyle)]], colWidths=[frame_width])
            table.setStyle(TableStyle([
                ("BACKGROUND", (0, 0), (-1, -1), colors.HexColor("#ecfdf5")),
                ("LINEBEFORE", (0, 0), (0, -1), 2.5, colors.HexColor("#10b981")),
                ("LEFTPADDING", (0, 0), (-1, -1), 10), ("RIGHTPADDING", (0, 0), (-1, -1), 10),
                ("TOPPADDING", (0, 0), (-1, -1), 7), ("BOTTOMPADDING", (0, 0), (-1, -1), 7),
            ]))
            story.append(table)
            story.append(Spacer(1, 8))
        elif block.kind == "image":
            from reportlab.platypus import Image
            file = DOCS / block.src
            if file.exists():
                picture = Image(str(file))
                scale = frame_width / picture.imageWidth
                picture.drawWidth, picture.drawHeight = frame_width, picture.imageHeight * scale
                story.append(picture)
                story.append(Paragraph(inline_pdf(block.text), captionStyle))
                story.append(Spacer(1, 8))
        elif block.kind == "rule":
            story.append(Spacer(1, 4))
            story.append(HRFlowable(width="100%", thickness=0.6, color=colors.HexColor("#e2e8f0")))
            story.append(Spacer(1, 6))
        elif block.kind == "table" and block.rows:
            header, *rest = block.rows
            columns = max(len(r) for r in block.rows)
            data = []
            for index, row in enumerate(block.rows):
                padded = list(row) + [""] * (columns - len(row))
                style = cellHead if index == 0 else cell
                data.append([Paragraph(inline_pdf(c), style) for c in padded])
            # 첫 열은 항목명이 오는 경우가 많아 조금 넓게 잡는다.
            weights = [1.35] + [1.0] * (columns - 1)
            total = sum(weights)
            widths = [frame_width * w / total for w in weights]
            table = Table(data, colWidths=widths, repeatRows=1)
            table.setStyle(TableStyle([
                ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#f1f5f9")),
                ("GRID", (0, 0), (-1, -1), 0.4, colors.HexColor("#e2e8f0")),
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("LEFTPADDING", (0, 0), (-1, -1), 5), ("RIGHTPADDING", (0, 0), (-1, -1), 5),
                ("TOPPADDING", (0, 0), (-1, -1), 4), ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
                ("ROWBACKGROUNDS", (0, 1), (-1, -1),
                 [colors.white, colors.HexColor("#fafafa")]),
            ]))
            story.append(table)
            story.append(Spacer(1, 9))
    _ = (KeepTogether, PageBreak)

    def footer(canvas, document):
        canvas.saveState()
        canvas.setFont("NanumGothic", 7.5)
        canvas.setFillColor(muted)
        canvas.drawString(16 * mm, 10 * mm, "SeatOn · scripts/build-docs.py 로 Markdown 원본에서 생성")
        canvas.drawRightString(A4[0] - 16 * mm, 10 * mm, str(canvas.getPageNumber()))
        canvas.restoreState()

    SimpleDocTemplate(
        str(path), pagesize=A4,
        leftMargin=16 * mm, rightMargin=16 * mm, topMargin=16 * mm, bottomMargin=18 * mm,
        title=doc.title, author="SeatOn",
    ).build(story, onFirstPage=footer, onLaterPages=footer)


def main() -> None:
    names = sys.argv[1:] or list(TARGETS)
    for name in names:
        source = DOCS / f"{name}.md"
        if not source.exists():
            raise SystemExit(f"원본을 찾을 수 없습니다: {source}")
        doc = parse_markdown(source.read_text(encoding="utf-8"))
        title = TARGETS.get(name, doc.title or name)
        (DOCS / f"{name}.html").write_text(render_html(doc, name, title), encoding="utf-8")
        if name in SHARED_PDF:
            print(f"{name}: html 생성 (블록 {len(doc.blocks)}개) — pdf 는 md2pdf.mjs 로 굽는다 (README 문서 산출물 참고)")
            continue
        render_pdf(doc, DOCS / f"{name}.pdf")
        print(f"{name}: html + pdf 생성 (블록 {len(doc.blocks)}개)")


if __name__ == "__main__":
    main()
