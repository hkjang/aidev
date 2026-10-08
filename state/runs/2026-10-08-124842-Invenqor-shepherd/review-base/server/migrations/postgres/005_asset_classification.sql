-- Collected inventory arrives as flat records with no business context: every
-- asset landed on the schema defaults ('other' environment, 'normal'
-- criticality, no owner) and relationships could only be typed in by hand. This
-- migration adds the two things a CMDB needs to make that automatic and
-- auditable: an ordered rule set that classifies, and provenance on both the
-- classification and every derived relationship.

CREATE TABLE asset_classification_rules (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    -- Lower runs first. Later rules can match on what earlier ones assigned,
    -- which is what lets "production hosts are high criticality" work.
    priority INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    -- A system rule ships with the product: it can be disabled or reprioritised
    -- but not deleted, so an upgrade can keep the taxonomy coherent.
    system_rule BOOLEAN NOT NULL DEFAULT FALSE,
    match_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    assign_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 0.8,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID REFERENCES users(id)
);

CREATE INDEX asset_classification_rules_order_idx
    ON asset_classification_rules(enabled, priority, name);

-- Provenance: which rules produced this asset's classification, how sure they
-- were, and which fields a human has taken over. Manual values must survive
-- every later automatic pass.
ALTER TABLE assets
    ADD COLUMN classification_source TEXT NOT NULL DEFAULT '';
ALTER TABLE assets
    ADD COLUMN classification_confidence DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE assets
    ADD COLUMN classification_rules_json JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE assets
    ADD COLUMN manual_fields_json JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE assets
    ADD COLUMN classified_at TIMESTAMPTZ;
ALTER TABLE assets
    ADD COLUMN tags_json JSONB NOT NULL DEFAULT '[]'::jsonb;

-- A derived relationship has to say how it was derived and whether a human has
-- accepted it, or nobody can trust the graph it draws.
ALTER TABLE asset_relations
    ADD COLUMN derivation TEXT NOT NULL DEFAULT '';
ALTER TABLE asset_relations
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE asset_relations
    ADD COLUMN reviewed_at TIMESTAMPTZ;
ALTER TABLE asset_relations
    ADD COLUMN reviewed_by UUID REFERENCES users(id);

CREATE INDEX asset_relations_status_idx
    ON asset_relations(status, relation_type);

INSERT INTO asset_classification_rules(
    id, name, description, priority, system_rule, match_json, assign_json,
    confidence
) VALUES
-- 1. Collector category to canonical type. These are facts, not guesses.
('20000000-0000-0000-0000-000000000001', '수집 범주 → 자산 유형',
 'Agent 수집 범주를 표준 자산 유형으로 정규화합니다.', 10, TRUE,
 '{"categories":["system"]}',
 '{"type":"host","relate_to_host":false}', 1.0),
('20000000-0000-0000-0000-000000000002', '서비스 유형',
 'init 관리 단위를 서비스 자산으로 분류합니다. 호스트 관계는 아래 역할 규칙이 붙은 서비스에만 만듭니다.', 10, TRUE,
 '{"categories":["service"]}',
 '{"type":"service","relate_to_host":false}', 1.0),
('20000000-0000-0000-0000-000000000003', '컨테이너 환경',
 '컨테이너 런타임 표식을 컨테이너 플랫폼 자산으로 분류합니다.', 10, TRUE,
 '{"categories":["container.environment","container"]}',
 '{"type":"container_platform","relate_to_host":true,"relation":"runs_on"}', 1.0),
('20000000-0000-0000-0000-000000000004', '네트워크 인터페이스',
 '인터페이스는 호스트의 구성 요소로 연결합니다.', 10, TRUE,
 '{"categories":["network.interface"]}',
 '{"type":"network_interface","relate_to_host":true,"relation":"part_of"}', 1.0),
('20000000-0000-0000-0000-000000000005', '파일시스템 볼륨',
 '마운트된 파일시스템을 스토리지 볼륨으로 분류합니다.', 10, TRUE,
 '{"categories":["hardware.filesystem"]}',
 '{"type":"storage_volume","relate_to_host":true,"relation":"attached_to"}', 1.0),
('20000000-0000-0000-0000-000000000006', '설치 소프트웨어',
 '패키지는 소프트웨어 자산으로 분류합니다. 개별 OS 패키지까지 관계를 만들면 그래프를 읽을 수 없으므로 기본적으로 관계는 만들지 않습니다.',
 10, TRUE,
 '{"categories":["software.package"]}',
 '{"type":"software","relate_to_host":false}', 1.0),
('20000000-0000-0000-0000-000000000008', '하드웨어 구성요소',
 'CPU와 메모리는 호스트의 하드웨어 구성요소로 연결합니다. 호스트마다 하나씩이라 그래프를 어지럽히지 않습니다.',
 10, TRUE,
 '{"categories":["hardware.cpu","hardware.memory"]}',
 '{"type":"hardware_component","relate_to_host":true,"relation":"part_of"}', 1.0),
('20000000-0000-0000-0000-000000000009', '네트워크 구성',
 '경로·DNS 구성은 호스트의 구성요소로 연결합니다.', 10, TRUE,
 '{"categories":["network.configuration"]}',
 '{"type":"network_configuration","relate_to_host":true,"relation":"part_of"}', 1.0),
('20000000-0000-0000-0000-00000000000a', '프로세스',
 '프로세스는 수집 시점의 상태이므로 자산 유형만 정규화하고 관계는 만들지 않습니다.',
 10, TRUE,
 '{"categories":["process"]}',
 '{"type":"process","relate_to_host":false}', 1.0),
('20000000-0000-0000-0000-000000000007', '계정',
 '로컬 계정을 계정 자산으로 분류합니다. 계정 수가 많아 관계는 만들지 않습니다.', 10, TRUE,
 '{"categories":["account.user","account"]}',
 '{"type":"account","relate_to_host":false}', 1.0),

-- 2. Software roles. A curated catalogue is what keeps the dependency graph
--    readable: middleware and databases earn a host relationship, a font
--    package does not.
('20000000-0000-0000-0000-000000000010', '데이터베이스 엔진',
 '알려진 데이터베이스 엔진을 database 유형으로 승격하고 호스트 관계를 만듭니다.', 40, TRUE,
 '{"categories":["service","software.package"],"name_patterns":["postgres*","*mysqld*","mariadb*","mongod*","redis*","*oracle*","*mssql*"]}',
 '{"type":"database","tags":["data-tier"],"relate_to_host":true,"relation":"runs_on"}', 0.9),
('20000000-0000-0000-0000-000000000011', '웹·프록시 계층',
 '웹 서버와 리버스 프록시를 web-tier 태그와 함께 서비스로 유지합니다.', 40, TRUE,
 '{"categories":["service","software.package"],"name_patterns":["nginx*","httpd*","apache2*","haproxy*","envoy*","traefik*","caddy*"]}',
 '{"type":"service","tags":["web-tier"],"relate_to_host":true,"relation":"runs_on"}', 0.9),
('20000000-0000-0000-0000-000000000012', '컨테이너 런타임',
 '컨테이너 런타임과 orchestrator 구성요소를 표시합니다.', 40, TRUE,
 '{"categories":["service","software.package"],"name_patterns":["docker*","containerd*","podman*","kubelet*","cri-o*"]}',
 '{"type":"container_platform","tags":["platform"],"relate_to_host":true,"relation":"runs_on"}', 0.9),
('20000000-0000-0000-0000-000000000013', '메시지·큐 계층',
 '메시지 브로커를 middleware 태그와 함께 표시합니다.', 40, TRUE,
 '{"categories":["service","software.package"],"name_patterns":["kafka*","rabbitmq*","activemq*","nats*","zookeeper*"]}',
 '{"type":"service","tags":["middleware"],"relate_to_host":true,"relation":"runs_on"}', 0.9),

-- 3. Environment from the host name. Every site names hosts, and the name is
--    the only environment signal the collector can see.
('20000000-0000-0000-0000-000000000020', '운영 환경 이름 규칙',
 '호스트 이름 토큰에 prd/prod/live가 있으면 production으로 봅니다. 부분 문자열이 아니라 구분자로 나눈 토큰을 비교하므로 postgresql의 stg 같은 오탐이 없습니다.',
 60, TRUE,
 '{"categories":["system","enrollment"],"name_tokens":["prd","prod","production","live"]}',
 '{"environment":"production"}', 0.6),
('20000000-0000-0000-0000-000000000021', '스테이징 이름 규칙',
 '호스트 이름 토큰에 stg/stage가 있으면 staging으로 봅니다.', 60, TRUE,
 '{"categories":["system","enrollment"],"name_tokens":["stg","stage","staging"]}',
 '{"environment":"staging"}', 0.6),
('20000000-0000-0000-0000-000000000022', 'QA 이름 규칙',
 '호스트 이름 토큰에 qa/uat가 있으면 qa로 봅니다.', 60, TRUE,
 '{"categories":["system","enrollment"],"name_tokens":["qa","uat"]}',
 '{"environment":"qa"}', 0.6),
('20000000-0000-0000-0000-000000000023', '테스트 이름 규칙',
 '호스트 이름 토큰에 test/tst가 있으면 test로 봅니다.', 60, TRUE,
 '{"categories":["system","enrollment"],"name_tokens":["test","tst"]}',
 '{"environment":"test"}', 0.6),
('20000000-0000-0000-0000-000000000024', '개발 이름 규칙',
 '호스트 이름 토큰에 dev가 있으면 development로 봅니다.', 60, TRUE,
 '{"categories":["system","enrollment"],"name_tokens":["dev","develop","development"]}',
 '{"environment":"development"}', 0.6),

-- 4. Criticality follows from environment and role, which is why these run last.
('20000000-0000-0000-0000-000000000030', '운영 데이터 계층 중요도',
 '운영 환경의 데이터베이스는 치명으로 분류합니다.', 80, TRUE,
 '{"environments":["production"],"types":["database"]}',
 '{"criticality":"critical"}', 0.7),
('20000000-0000-0000-0000-000000000031', '운영 호스트 중요도',
 '운영 환경의 호스트와 서비스는 높음으로 분류합니다.', 82, TRUE,
 '{"environments":["production"],"types":["host","service","container_platform"]}',
 '{"criticality":"high"}', 0.7),
('20000000-0000-0000-0000-000000000032', '비운영 중요도',
 '개발·테스트 자산은 낮음으로 분류합니다.', 84, TRUE,
 '{"environments":["development","qa","test"]}',
 '{"criticality":"low"}', 0.7);
