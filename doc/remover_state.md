# REMOVER STATES

## v1.0.0.qr2 / 2020-01-22
```plantuml
title cfm_v1.0.0.qr2_remover_State
hide empty description

[*]--> S0
S0 : 무한 반복
S0 -> S1
S1 : grade info, hitcount history 파일로부터\n파일 등급, 크기, 서버 위치 정보가 들어있는 파일 meta 목록 생성\n여러 server에 존재하는 파일 meta 목록 생성
note right
	파일 등급 : grade info 파일에서 구함
	파일 크기 : hitcount history 파일에서 구함
	서버 위치 : 서버가 존재하는 서버 ip 목록, hitcount history 파일에서 구함

	서버 위치 정보를 이용하여 여러 server에 존재하는 파일 목록 생성
end note

S1 --> S7 : 파일 parsing error 발생
S7 : sleep N초(default:5)
S7 --> S0
S1 --> S2 : parsing 성공
S2 : 급hit 상승 파일 목록 구함
S2 --> S3
S3 : server별로 존재하는 파일 meta 목록 구함
S3 --> S4
S4 : 여러 server에 존재하는 파일에 대한 삭제 요청
S4 -> S5
S5 : disk 사용량 limit를 넘은 server 조사
S5 --> S6
S6 : disk 사용량 limit를 넘은 server에 대해서 disk 삭제 요청
S6 --> S7
```

```plantuml
title cfm_v1.0.0.qr2_remover_delete_duplicated_files_State
hide empty description
state S4 {
S4 : 여러 server에 존재하는 파일에 대한 삭제 요청
[*] -> S40
S40 : 여러 server에 존재하는 파일 meta 목록 update
note right
	server별로 존재하는 파일 meta 목록을 가지고
	cross check하여 update
end note

S40 --> S41
S41 : 여러 server에 존재하는 파일이\n하나의 서버에만 존재할 때까지 삭제 요청 수행
S41 --> S42
S42 : 특정 prefix로 시작하는 파일이면 제외 (광고 파일 제외)
S42 --> S43
S43 : source path에 없는 파일이면 제외 (SAN에 없는 파일 제외)
S43 --> S44
S43 : 삭제 요청
S44 --> S41 : 처리할 파일이 남아있으면
S44 --> [*] : 처리할 파일이 없으면
}
```

```plantuml
title cfm_v1.0.0.qr2_remover_delete_files_for_disk_free_State
hide empty description
state S6 {
S6 : disk 사용량 limit를 넘은 server 들에 대해서 수행
[*] -> S60
S60 : disk 사용량 limit를 넘은 server에 대해서 수행
S60 --> S61
S61 : server별로 지워야 할 파일 구하기
S61 --> S62
S62 : 특정 prefix로 시작하는 파일이면 제외 (광고 파일 제외)
S62 --> S63
S63 : source path 에 없는 파일이면 제외 (SAN에 없는 파일 제외)
S63 --> S64
S64 : 급 hit 상승 파일이면 제외
S64 --> S61 : 처리할 파일이 남아있으면
S64 --> S65 : 처리할 파일이 없으면
S65 : 지워야할 파일 목록을 등급이 큰 순서로 정렬
S65 --> S67
S67 : disk 여유 사용량이 확보될 때까지\n지워야할 파일 목록의 파일에 대해서\n 서버에 삭제 요청
S67 --> S60 : 처리할 sever가 남아있으면
S67 --> [*] : 처리할 server가 없으면
}
```


## v1.0.0.qr1 / 2019-11-12
```plantuml
title cfm_v1.0.0.qr1_remover_State
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
