AppStore v2.6.1

소유자의 내 앱 목록과 내 홈에서 초안·검토 대기·반려·보관 앱이 게시된 앱과 똑같이 보여, 어느 앱이 어떤 상태인지 카드만으로는 알 수 없던 문제를 고친 patch 릴리스입니다.

- `AppCard`는 공개 카탈로그용이라 status를 그리지 않는데 `/my/apps`와 내 홈이 같은 컴포넌트를 재사용해, 소유자의 초안·검토 대기·반려·보관 앱이 모두 같은 모양이었습니다. 내 홈의 숫자 카드는 '몇 개'만 말하고 '어느 것'인지는 말하지 않았습니다.
- `AppCard`에 `showStatus` prop을 두어 소유자 화면 두 곳만 배지를 켭니다. 공개 카탈로그는 게시된 앱만 있으므로 그대로이고, 캡처도 바이트 동일합니다.
- admin-pages에 있던 `APP_STATUSES`·`AppStatusBadge`를 `features/apps/app-status.tsx`로 옮겨 관리 콘솔과 소유자 화면이 같은 라벨과 톤을 씁니다.
- e2e mock의 `/me/apps`에 review가 실린 반려 앱(Release Radar)을 더하고 my-app-edit 캡처를 그 앱으로 바꿔, my-apps·my-home·my-app-edit 캡처 6장이 상태 배지와 반려 안내를 실제로 담습니다. 사용자 가이드 2.4·3.8·3.9·4.1의 "카드에는 배지가 없다"는 우회 안내를 고치고 PDF를 다시 구웠습니다.
- Schema 변경이 없어 기존 설치는 image만 교체하면 됩니다.
