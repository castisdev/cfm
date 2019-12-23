# REMOVER STATES

## ASIS:
```plantuml
hide empty description

[*]--> S0
S0 : 무한 반복\ngrade info, hitcount history 파일 parsing 상태 초기화
S0 --> S1
S1 : 모든 server의 disk 사용량 정보 구함\nerror난 경우, disk 크기, 사용량은 초기값으로 구해짐
S1 --> S2
S2 : 모든 server의 disk 사용량 정보
S2 --> S5 : 더 이상 처리할 정보가 없는 경우
S2 --> S3 : 처리할 disk 사용량 정보가 있는 경우
S3 : disk 사용량 limit 검사
S3 --> S4 : limit 넘은 경우
S3 --> S2 : limit 넘지 않은 경우\n(disk 사용량을 구하지 못한 경우도 포함됨)
S4 : grade info, hitcount history 파일로부터 파일 등급, 크기 정보가 들어있는 목록 생성,\n이미 파일 parsing 한 상태이면 다시 parsing 하지 않음
S4 --> S5 : 파일 parsing error 발생
S5 : sleep 3초
S5 --> S0
S4 --> S6 : parsing 성공
S6 : server 정보 구하기\nserver 의 파일 목록 구함
S6 --> S5 : error 발생
S6 --> S7 : 성공
S7 : 광고 파일 제외
S7 --> S8
S8 : 등급, 크기 정보에서 파일 등급 구함
S8 --> S9
S9 : 등급, 크기 정보에서 파일 크기 구함
S9 --> S10
S10 : 낮은 등급 순으로 정렬
S10 --> S11
S11 : 파일 크기 정보없는 파일 제외
S11 --> S12
S12 : Source path에 없는 파일 제외
S12 --> S13
S13 : 남은 파일 목록을 가지고 삭제 파일 대상 목록 만들기\ndisk 사용량 limit 넘은 크기 만큼의 파일 목록 만들기
S13 --> S14
S14 : 서버에 삭제 대상 목록에 대해서 삭제 요청
S14--> S2
```

## TOBE:
```plantuml
hide empty description

[*]--> S0
S0 : 무한 반복
S0 -> S1
S1 : grade info, hitcount history 파일로부터 파일 등급, 크기 정보가 들어있는 목록 생성

S1 --> S3 : 파일 parsing error 발생
S3 : sleep 3초
S3 --> S0
S1 --> S2 : parsing 성공
S2 : 급hit 상승 파일 목록 구함
S2 --> S4
S4 : 모든 server의 disk 사용량 정보 구함
S4 --> S5
S5 : 모든 server에 대해서 처리
S5 --> S3 : 처리할 서버가 없는 경우
S5 --> S6 : 처리할 서버가 있는 경우
S6 : disk 사용량 limit 검사
S6 --> S7 : limit 넘은 경우
S6 --> S8 : limit 넘지 않은 경우


```
