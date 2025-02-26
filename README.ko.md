# Cachembed

OpenAI 임베딩 API 요청을 위한 경량 캐싱 프록시입니다.

## 개요

Cachembed는 OpenAI 임베딩 API 결과를 캐싱하여 중복 요청을 줄이고 비용을 최소화하는 프록시 서버입니다. 기본적으로 SQLite와 PostgreSQL을 저장 backend로 지원합니다.

## 기능

- 임베딩 결과를 SQLite 또는 PostgreSQL에 캐시
- OpenAI API로 요청을 프록시 (기본 URL: https://api.openai.com/v1/embeddings)
- 정규 표현식 패턴을 통한 API 키 검증 지원
- 허용된 임베딩 모델로 사용 제한
- 데이터베이스 마이그레이션 지원
- 환경 변수를 통한 설정 가능

## 요구 사항

* Ruby 3.4.1 이상
* Rails 8.0.1 이상
* SQLite3 또는 PostgreSQL

## 설치

저장소를 클론하고 종속성을 설치합니다:

    git clone https://github.com/your-username/cachembed-rails
    cd cachembed-rails
    # PostgreSQL을 사용하려면 다음을 실행:
    bundle install --with=postgresql
    # SQLite를 사용하려면 다음을 실행:
    bundle install

## 설정

데이터베이스를 생성하고 마이그레이션합니다:

    bin/setup --skip=server

## 구성

이 환경 변수를 사용하여 애플리케이션을 구성합니다:

| 환경 변수 | 설명 | 기본값 |
|------------|------|---------|
| CACHEMBED_UPSTREAM_URL | OpenAI 임베딩 API 엔드포인트 | https://api.openai.com/v1/embeddings |
| CACHEMBED_ALLOWED_MODELS | 허용된 모델의 콤마로 구분된 목록 | text-embedding-3-small,text-embedding-3-large,text-embedding-ada-002 |
| CACHEMBED_API_KEY_PATTERN | API 키 검증을 위한 정규 표현식 패턴 | ^sk-[a-zA-Z0-9_-]+$ |
| DATABASE_URL | 데이터베이스 연결 문자열 | config/database.yml에 따라 다름 |

## 사용법

### 서버 시작하기

개발 환경:

    rails server

프로덕션 환경:

    RAILS_ENV=production rails server

### API 엔드포인트

서버는 다음과 같은 엔드포인트를 제공합니다:

- POST `/v1/embeddings`: 캐싱과 함께 OpenAI의 임베딩 API로 요청을 프록시

예시 요청:

    curl -X POST http://localhost:3000/v1/embeddings \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer sk-your-api-key" \
      -d '{
        "input": "여기에 당신의 텍스트를 입력하세요",
        "model": "text-embedding-3-small"
      }'

## 라이선스

MIT 라이선스

## 기여

풀 리퀘스트는 환영합니다! 버그를 발견하거나 기능 요청을 원하시면 이슈를 열어주세요.

## TODO

- LRU 캐시 (요청 로그 포함) 및 오래된 캐시 항목에 대한 가비지 컬렉션