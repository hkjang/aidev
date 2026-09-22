# 배포 파일 안내

기본 배포는 저장소 루트의 `compose.yaml`을 사용합니다. PostgreSQL은 이미 운영 중인 인스턴스를 사용하거나 별도로 준비합니다. 릴리즈에는 madi 서비스 이미지 하나만 포함됩니다.

## Kubernetes

`kubernetes.yaml`은 단일 복제본, 영구 볼륨, 상태 점검, 비 root 실행을 포함한 시작 예제입니다. 운영 클러스터의 StorageClass, TLS ingress, 리소스 및 네임스페이스 정책에 맞게 조정하세요. 로컬 파일 스토리지를 사용하므로 여러 복제본으로 바로 확장하지 마세요.

기본 리소스는 CPU 요청 100m/한도 2, 메모리 요청 512MiB/한도 4GiB이며 `/tmp`에 디스크 기반 2GiB `emptyDir`를 제공합니다. 가져오기·논리 복원 중 임시 데이터가 생기므로 노드의 ephemeral-storage 여유도 확보하세요. 기존 64MiB 임시 공간으로는 최대 크기 가져오기/복원을 처리할 수 없습니다. 여러 작업을 동시에 실행하는 경우 관리자 작업 설정의 동시성과 리소스를 함께 조정합니다.

1. 암호와 DSN을 입력한 `.env` 파일을 준비합니다. 키 이름은 `.env.example`의 네 개만 사용합니다.
2. Secret을 생성합니다. `.env`는 별도 비밀 관리 저장소에서 보관합니다.

```sh
kubectl create secret generic madi-secrets --from-env-file=.env
```

3. 모든 배포 대상 노드의 컨테이너 런타임에 `madi:v0.3.0` 이미지를 반입합니다. Docker 노드는 `docker load`, containerd 노드는 해당 배포 환경에서 지정한 이미지 import 명령을 사용합니다. 외부 레지스트리를 참조하지 않도록 예제의 `imagePullPolicy: Never`를 유지합니다.
4. 배포합니다.

```sh
kubectl apply -f deploy/kubernetes.yaml
kubectl rollout status deployment/madi
kubectl port-forward service/madi 8080:8080
```

처음 로그인한 뒤 관리자의 시스템 설정에서 실제 서비스 URL을 지정합니다. Keycloak의 리디렉션 URI와 서비스 URL은 동일한 HTTPS 출처를 사용해야 합니다. PostgreSQL TLS 인증서가 사설 CA를 사용하면 컨테이너 신뢰 저장소에 CA를 배포하는 사내 표준을 적용하세요. CA 파일은 별도 서비스 환경변수 추가 없이 마운트 또는 이미지 신뢰 저장소 구성으로 관리할 수 있습니다.

자세한 운영 절차는 [오프라인 운영 가이드](../docs/offline-guide.md)를 참조하세요.
