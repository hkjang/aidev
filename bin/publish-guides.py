#!/usr/bin/env python3
"""모아 둔 가이드를 AppStore 의 앱마다 첨부한다 — 바뀐 것만.

사람들이 앱을 찾는 곳은 AppStore 다. 가이드가 저장소 안에만 있으면 그 저장소를
아는 사람만 읽는다. bin/collect-guides.py 가 이미 모든 서비스의 가이드를
docs/guides/ 아래로 모아 두므로, 여기서는 그것을 앱마다 밀어 넣는다.

**가이드 문서만 건드린다.** 앱을 새로 만들지 않고, 이름·설명·아이콘 같은 앱의 다른
항목은 읽지도 고치지도 않는다. AppStore 에 없는 앱은 그냥 건너뛴다.

바뀐 것만 올린다. 목록 응답의 checksum 과 파일의 checksum 을 견주어 같으면 넘어간다.
그러지 않으면 30분마다 같은 PDF 를 다시 밀어 넣게 된다.

설정은 ~/.auto-improve/appstore.env 에 둔다(저장소 밖):
    APPSTORE_URL=https://appstore.intra
    APPSTORE_API_KEY=...
"""
from __future__ import annotations

import hashlib
import json
import mimetypes
import os
import sys
import urllib.error
import urllib.request
import uuid
from pathlib import Path

AIDEV = Path(__file__).resolve().parent.parent
GUIDES = AIDEV / "docs" / "guides"
ENV = Path.home() / ".auto-improve" / "appstore.env"
TIMEOUT = 30
# 올릴 문서: 파일 이름 → (kind, 화면에 보일 이름)
WANTED = {
    "USER_GUIDE.pdf": ("user_guide", "사용자 가이드"),
    "ADMIN_GUIDE.pdf": ("admin_guide", "관리자 가이드"),
}


def settings() -> tuple[str, str]:
    values: dict[str, str] = {}
    if ENV.is_file():
        for line in ENV.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, _, value = line.partition("=")
                values[key.strip()] = value.strip()
    url = os.environ.get("APPSTORE_URL") or values.get("APPSTORE_URL", "")
    key = os.environ.get("APPSTORE_API_KEY") or values.get("APPSTORE_API_KEY", "")
    return url.rstrip("/"), key


def call(method: str, url: str, key: str, body: bytes | None = None, content_type: str = ""):
    request = urllib.request.Request(url, data=body, method=method)
    request.add_header("Authorization", "Bearer " + key)
    if content_type:
        request.add_header("Content-Type", content_type)
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            raw = response.read()
            return response.status, (json.loads(raw) if raw else None)
    except urllib.error.HTTPError as error:
        return error.code, None
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError):
        return 0, None


def multipart(path: Path, kind: str, title: str) -> tuple[bytes, str]:
    boundary = "----aidev" + uuid.uuid4().hex
    mime = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    parts: list[bytes] = []
    for name, value in (("kind", kind), ("title", title)):
        parts.append(
            f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n'.encode()
        )
    parts.append(
        f'--{boundary}\r\nContent-Disposition: form-data; name="file"; filename="{path.name}"\r\n'
        f"Content-Type: {mime}\r\n\r\n".encode()
    )
    parts.append(path.read_bytes())
    parts.append(f"\r\n--{boundary}--\r\n".encode())
    return b"".join(parts), "multipart/form-data; boundary=" + boundary


def main() -> int:
    base, key = settings()
    if not base or not key:
        print("AppStore 설정이 없어 건너뜁니다 (~/.auto-improve/appstore.env)")
        return 0
    if not GUIDES.is_dir():
        print("모아 둔 가이드가 없습니다 — bin/collect-guides.py 를 먼저 돌리세요")
        return 0

    pushed = skipped = missing = 0
    for project in sorted(p for p in GUIDES.iterdir() if p.is_dir()):
        slug = project.name.lower()
        files = {name: project / name for name in WANTED if (project / name).is_file()}
        if not files:
            continue
        status, listed = call("GET", f"{base}/api/v1/apps/{slug}/documents", key)
        if status == 404:
            missing += 1
            continue
        if status != 200 or listed is None:
            print(f"{project.name}: 목록을 읽지 못했습니다 (HTTP {status})")
            continue
        have = {item.get("kind"): item.get("checksum") for item in listed}
        for name, path in files.items():
            kind, title = WANTED[name]
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            if have.get(kind) == digest:
                skipped += 1
                continue
            body, content_type = multipart(path, kind, title)
            status, _ = call("POST", f"{base}/api/v1/apps/{slug}/documents", key, body, content_type)
            if status in (200, 201):
                print(f"{project.name}: {title} 올림 ({path.stat().st_size // 1024}KB)")
                pushed += 1
            else:
                print(f"{project.name}: {title} 올리지 못했습니다 (HTTP {status})")
    print(f"올림 {pushed} · 그대로 {skipped} · AppStore 에 없는 앱 {missing}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
