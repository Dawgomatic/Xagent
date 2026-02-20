# RAG 평가 리포트 v2 — bge-m3 + BM25 하이브리드 검색

**날짜**: 2026-02-17
**모델**: `bge-m3` (기존 `nomic-embed-text` → `qwen3:8b` → `bge-m3`)
**검색**: 하이브리드 (0.6 × vector cosine + 0.4 × BM25)

## 결과 요약

| 지표 | v1 (nomic/qwen3) | v2 (bge-m3 + BM25) |
|---|---|---|
| Top-1 HIGH 비율 | 낮음 (동일 문서 반복) | **10/10 (100%)** |
| HIGH 결과 | - | 22/30 (73%) |
| MED 결과 | - | 6/30 (20%) |
| LOW 결과 | - | 2/30 (7%) |

## 변경 사항

1. **임베딩 모델**: `nomic-embed-text` → `bge-m3` (1.2GB, 다국어 특화)
2. **BM25 하이브리드**: 키워드 exact match 보완, 한국어 bigram 토크나이저
3. **API 엔드포인트**: `/api/embed` 전용 사용

## 쿼리별 Top-1 결과

| # | 쿼리 | Top-1 Score | Top-1 타입 | 적합도 |
|---|---|---|---|---|
| Q1 | TIPS 합격하려면? | 0.7060 | success_case | ✅ HIGH |
| Q2 | 초기창업패키지 심사기준 | 0.8431 | criteria | ✅ HIGH |
| Q3 | 예비창업패키지 합격 후기 | 0.8217 | success_case | ✅ HIGH |
| Q4 | 2026년 정부 창업지원사업 | 0.8277 | success_case | ✅ HIGH |
| Q5 | 벤처투자 현황 | 0.8011 | vc_investment | ✅ HIGH |
| Q6 | TIPS 운영사별 차이점 | 0.7755 | gov_program | ✅ HIGH |
| Q7 | 창업도약패키지 신청 자격 | 0.7896 | criteria | ✅ HIGH |
| Q8 | 청년창업사관학교 합격 노하우 | 0.7906 | success_case | ✅ HIGH |
| Q9 | 시리즈A 투자 받으려면 | 0.7421 | success_case | ✅ HIGH |
| Q10 | 정부지원사업 가점 방법 | 0.7633 | criteria | ✅ HIGH |
