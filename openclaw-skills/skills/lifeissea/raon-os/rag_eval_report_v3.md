# RAG Eval Report v3 — 2026.02.17

## 변경사항
- **kiwipiepy 형태소 분석** BM25 토크나이저 교체 (bigram fallback 유지)
- **통합공고 PDF 537건** 파싱 → `unified_announcement_2026.jsonl`
- **웹 성공사례 20건** 수집 → `success_cases_web.jsonl`
- 벡터 저장소: 151건 → **707건** (4.7x 증가)

## 평가 결과
| 지표 | v2 | v3 |
|------|-----|-----|
| 벡터 저장소 | 151건 | 707건 |
| Top-1 HIGH | - | **10/10 (100%)** |
| 전체 HIGH | - | **25/30 (83%)** |
| MED | - | 5/30 (17%) |
| LOW | - | 0/30 (0%) |

## 쿼리별 Top-1 결과
| # | 쿼리 | Top-1 Score | Relevance | Type |
|---|-------|------------|-----------|------|
| Q1 | TIPS 합격하려면? | 0.7679 | HIGH | success_case |
| Q2 | 초기창업패키지 심사기준 | 0.8431 | HIGH | criteria |
| Q3 | 예비창업패키지 합격 후기 | 0.8167 | HIGH | success_case |
| Q4 | 2026년 창업지원사업 | 0.8277 | HIGH | success_case |
| Q5 | 벤처투자 현황 | 0.8011 | HIGH | vc_investment |
| Q6 | TIPS 운영사 차이점 | 0.7346 | HIGH | gov_program |
| Q7 | 창업도약패키지 신청자격 | 0.7868 | HIGH | gov_program |
| Q8 | 청년창업사관학교 노하우 | 0.8230 | HIGH | success_case |
| Q9 | 시리즈A 투자 | 0.7421 | HIGH | success_case |
| Q10 | 정부지원사업 가점 | 0.8105 | HIGH | success_case |

## 데이터 구성
| 파일 | 건수 | 설명 |
|------|------|------|
| criteria_supplement.jsonl | 35 | 심사기준 |
| gov_programs_supplement.jsonl | 37 | 정부 프로그램 |
| success_cases.jsonl | 59 | 성공사례 (수동) |
| success_cases_web.jsonl | 20 | 성공사례 (웹 수집) |
| unified_announcement_2026.jsonl | 537 | 2026 통합공고 |
| vc_investment.jsonl | 19 | VC 투자 현황 |

## BM25 개선
- kiwipiepy 형태소 분석: NNG, NNP, VV, VA, SL, SN, NNB, XR, MAG 품사만 추출
- 기존 bigram 방식은 fallback으로 유지 (kiwipiepy 미설치 환경 대응)
- hybrid search: vector 60% + BM25 40% 가중치
