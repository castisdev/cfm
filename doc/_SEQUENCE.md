v1.0.0.qr2 / 2020-01-22
===================

### cfm, cfw, DFS

- 배포 schedule과 task는 같은 의미로 사용됨
- cfw 와 heartbeat 추가
- 여러 서버에 배포된 파일을 한 서버에만 남기고 나머지에서는 제거하는 기능 추가

```plantuml

box "ADC/LSM"
participant cfm
end box

box "VOD"
participant cfw as "cfw"
end box

box "VOD(source)"
participant cfw2 as "cfw"
participant dfs as "DFS"
end box

group heartbeat-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfw: heartbeat 확인
    note right
      서버별 heartbeat 성공 여부 정보 생성
    end note
cfm->cfw2: heartbeat 확인
end
end

group file-remover-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note
cfm->cfw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cfw2: GET /files
cfm->cfm: file meta 에서 여러 서버에 존재하는(배포된) file list 추출
cfm->cfw: DELETE /files/c.mpg
    note right
        여러 서버에 존재하는 file 삭제
    end note

cfm->cfw: GET /df
    note right
        디스크 사용량 체크
    end note
cfm->cfw2: GET /df
cfm->cfw: DELETE /files/a.mpg
    note right
        디스크 사용량이 정해진 한계 사용량을 넘었을 때
        file 순위가 낮은 순으로 file 삭제
    end note
cfm->cfw2: DELETE /files/b.mpg
end
end

group task-manager-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note
cfm->cfm: check task queue
    note right
        배포 schedule 조회
    end note
cfm->cfm: clean task queue
    note right
        schedule 삭제
        - 완료되었거나 (status = done)
        - timeout ( mtime + 30분 =< current time )
        - heartbeat 성공 하지 않았거나
    end note
cfm->cfw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cfw2: GET /files
cfm->cfm: create tasks
    note right
        배포 schedule 생성
        - 순위가 높은 순서부터 조건 확인
        - task queue 에 없고
        - VOD 서버에 없고
        - SAN 에 존재하는 파일인 경우 선택
        - 모든 src 가 선택될 때까지 생성
    end note
end
end

group cfw
loop endless
autonumber 1 "<b>[0]"
cfw->cfw: 디스크 사용량 체크
cfw->cfm: GET /tasks
    note left
        디스크가 충분한 경우
        배포 schedule 조회
        자신의 배포 schedule 선택
    end note
cfw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 시작 전 schedule 상태 변경
        - status:ready -> status:working
    end note
cfw->dfs: file 다운로드 요청
cfw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 완료 후 schedule 상태 변경
        - status:working -> status:done
    end note
end
end
```
### tasker

- task 관리 모듈

```plantuml
title cfm_v1.0.0.qr2_tasker_State
hide empty description

[*]-> S0
S0 : 무한 반복
S0 --> S1
S1 : grade info, hitcount history 파일로부터\n파일 등급, 크기, 서버 위치 정보가 들어있는 파일 meta 목록 생성\n여러 server에 존재하는 파일 meta 목록 생성
note right
	파일 등급 : grade info 파일에서 구함
	파일 크기 : hitcount history 파일에서 구함
	서버 위치 : 서버가 존재하는 서버 ip 목록,
                     hitcount history 파일에서 구함

	서버 위치 정보를 이용하여 여러 server에
  존재하는 파일 목록 생성
end note
S1 --> S100 : 파일 parsing error 발생

S1 --> S2 : parsing 성공
S2 : 급hit 상승 파일 목록 구함

S2 --> S3
S3 : source, dest 서버의 heartbeat 결과 얻음

S3 --> S4
S4: task 정리
note right
	DONE task 정리
	TIMEOUT 계산해서 TIMEOUT된 task 정리
	Src 또는 Dest의 heartbeat 답을 구하지 못한 task 정리
end note

S4 --> S5
S5 : 배포에 사용 가능한\nsource 서버가 있는 지 체크
S5 --> S100 : 사용 가능한 source 서버가\n없는 경우

S5 --> S6
S6 : 배포에 사용 가능한\ndest 서버가 있는 지 체크
S6 -> S100 : 사용 가능한 dest 서버가\n없는 경우

S6 --> S7
S7 : 모든 dest 서버의 파일 목록 수집

S7 --> S8
S8 : 배포 대상 파일 목록 만들기

S8 --> S9
S9 : 배포 대상 파일 목록에 대해서\n배포 task 만들기

S9 --> S10
S10 : src 서버 선택

S10 --> S100 : 사용 가능한 source 서버가\n없는 경우

S10 --> S11
S11 : dest 서버 선택
note right
 dest 서버는 같은 서버가
 여러 번 선택될 수 있음
end note

S11 --> S12
S12 : 배포 task 생성

S12 --> S9 : 배포할 파일이 남은 경우

S12 --> S100 : 더 이상 배포할 피일이 없는 경우

S100 -> S0
S100: sleep N초(default:10초)
```

```plantuml
title cfm_v1.0.0.qr2_tasker_make_task_files_State
hide empty description

state S8 {
S8 : 배포 대상 파일 목록 만들기

S80: grade info, hitcount history로 만든\n파일 meta 목록에 급 hits 반영
S81: grade info, hitcount history로 만든\n파일 meta 목록에 대해서 수행
S82: dest 서버에 이미 있는 파일이면 제외
S83: source path에 없는 파일이면 제외 (SAN에 는 파일이면 제외)
S84: 특정 prefix로 시작하는 파일이면 제외 (광고 파일 제외)
S85: 배포에 사용 중인 파일이면 제외 (광고 파일 제외)
S86: 급 hits 높은 순으로 정렬\n등급값 낮은 순으로 정렬

[*] --> S80
S80 --> S81
S81 --> S82
S82 --> S83
S83 --> S84
S84 --> S85
S85 --> S86
S86 --> S81 : 처리할 파일이 남은 경우
S86 --> [*] : 더 이상 파일이 없는 경우
}
```

### remover State

- 파일 삭제 요청 모듈

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
S42 : 특정 prefix로 시작하는 파일일 때는 제외 (광고 파일 제외)
S42 --> S43
S43 : source path 에 없는 파일일 때는 제외 (SAN에 없는 파일 제외)
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
S62 : 특정 prefix로 시작하는 파일일 때는 제외 (광고 파일 제외)
S62 --> S63
S63 : source path 에 없는 파일일 때는 제외 (SAN에 없는 파일 제외)
S63 --> S64
S64 : 급 hit 상승 파일일 때는 제외
S64 --> S61 : 처리할 파일이 남아있으면
S64 --> S65 : 처리할 파일이 없으면
S65 : 지워야할 파일 목록을 등급이 큰 순서로 정렬
S65 --> S67
S67 : disk 여유 사용량이 확보될 때까지\n지워야할 파일 목록의 파일에 대해서\n 서버에 삭제 요청
S67 --> S60 : 처리할 sever가 남아있으면
S67 --> [*] : 처리할 server가 없으면
}
```



v1.0.0.qr1 / 2019-11-12
===================

### cfm, cfw, DFS

- 배포 schedule과 task는 같은 의미로 사용됨

```plantuml

box "ADC/LSM"
participant cfm
end box

box "VOD"
participant cw as "cfw"
end box

box "VOD(source)"
participant cw2 as "cfw"
participant dfs as "DFS"
end box

group file-remover-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note

cfm->cw: GET /df
    note right
        디스크 사용량 체크
    end note
cfm->cw2: GET /df
cfm->cw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cw2: GET /files
cfm->cw: DELETE /files/a.mpg
    note right
        디스크 사용량이 정해진 한계 사용량을 넘었을 때
        file 순위가 낮은 순으로 file 삭제
    end note
cfm->cw2: DELETE /files/b.mpg
end
end

group task-manager-thread
loop endless
autonumber 1 "<b>[0]"
cfm->cfm: file meta 정보 생성
    note right
        - .hitcount.history 에서 file size 추출
        - .grade.info 에서 file 순위 추출
    end note
cfm->cfm: check task queue
    note right
        배포 schedule 조회
    end note
cfm->cfm: clean task queue
    note right
        schedule 삭제
        - 완료되었거나 (status = done)
        - timeout ( mtime + 30분 =< current time )
    end note
cfm->cw: GET /files
    note right
        서버별 file list 확인
    end note
cfm->cw2: GET /files
cfm->cfm: create tasks
    note right
        배포 schedule 생성
        - 순위가 높은 순서부터 조건 확인
        - task queue 에 없고
        - VOD 서버에 없고
        - SAN 에 존재하는 파일인 경우 선택
        - 모든 src 가 선택될 때까지 생성
    end note
end
end

group cfw
loop endless
autonumber 1 "<b>[0]"
cw->cw: 디스크 사용량 체크
cw->cfm: GET /tasks
    note left
        디스크가 충분한 경우
        배포 schedule 조회
        자신의 배포 schedule 선택
    end note
cw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 시작 전 schedule 상태 변경
        - status:ready -> status:working
    end note
cw->dfs: file 다운로드 요청
cw->cfm: PATCH /tasks/${task_id}
    note left
        file 다운로드 완료 후 schedule 상태 변경
        - status:working -> status:done
    end note
end
end
```

### tasker

- task 관리 모듈

```plantuml
title cfm_v1.0.0.qr1_tasker_State
hide empty description

[*]-> S0
S0 : 무한 반복
S0 --> S1
S1: task 정리
note right
  DONE, TIMEOUT task 정리
end note
S1 --> S2
S2 : task 에서 사용 중(배포 중)인\nsource 서버가 있는 지 체크
S2 --> S3 : 사용 가능한 source 서버가\n없는 경우
S3 : sleep 5초
S3 -> S0

S2 --> S4
S4 : grade 정보, hitcount 정보를 이용하여\n파일 목록 생성
note right
 파일 등급 : grade info 파일에서 구함
 파일 크기 : hitcount 정보에서는 구함
end note

S4 --> S3 : error 가 발생하는 경우

S4 --> S5 : 정상 처리된 경우
S5 : 광고 파일은 배포 파일 목록에서 제외
S5 --> S6
S6 : 파일 목록을 높은 등급 순서로 정렬

S6 --> S7
S7 : dest 서버의 모든 파일 목록 수집

S7 --> S8
S8 : 급 hit 상승 파일 목록 생성
note left
LB eventlog 이용하여
N분 동안 X번 이상 특정 IP 에서
서비스된 파일 목록을 구함
end note

S8 --> S9
S9 : 급 hit 상승 파일 목록을\nhitcount 큰 순서로 정렬

S9 --> S10
S10 : 급 hit 상승 파일 목록에, \ngrade 정보, hitcount 정보로 만든 파일 목록을  함침

S10 --> S11
S11: 합친 배포 파일 목록으로 배포 task 만들기
note right
 사용 중이 아닌 source 서버 갯수만큼
 배포 task를 만듬
end note

S11 --> S12
S12 : 다음 파일 선택 (순서대로 선택)

S12 --> S13
S13 : 이미 배포 중인 파일이면 제외

S13 --> S14
S14 : dest 서버에 있는 파일이면 제외
S14 --> S15
S15 : source path에 없는 파일이면 제외 (SAN에 없는 파일이면 제외)
S15 --> S16
S16 : 사용 가능한 source가 있는 지 검사

S16 --> S17
S17 : source 서버 선택

S17 --> S18
S18: dest 서버 선택 (ring 구조체에 저장되어있음)
note right
 dest 서버는 같은 서버가
 여러 번 선택될 수 있음
end note

S18 --> S19
S19: 배포 task 생성

S16 --> S20: 사용 가능한 source 서버가\n없는 경우
S20 : sleep 60초
S20 --> S0

S19 --> S20 : 더 이상 배포할 피일이 없는 경우
S19 --> S11 : 배포할 파일이 남은 경우
```

### remover State

- 파일 삭제 요청 모듈

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

기존 모듈간의 Seqence Diagram
===================
* CenterFileManager, CenterFileWorker, DFS, vodSubAgent

```plantuml
box "ADC/LSM"
participant cm as "CenterFileManager"
participant dfs as "DFS"
end box

box "VOD"
participant cw1 as "CenterFileWorker-1"
participant dfs1 as "DFS-1"
participant va1 as "vodSubAgent-1"
end box

box "VOD(Source)"
participant cw2 as "CenterFileWorker-2"
participant dfs2 as "DFS-2"
participant va2 as "vodSubAgent-2"
end box

loop schedule file 발견할 때까지
cw1->cw1: scheule file polling
end

loop schedule file 발견할 때까지
cw2->cw2: scheule file polling
end

cm->cm: file meta 생성
cm->dfs1: file list 조회
cm->dfs2: file list 조회
cm->va2: vod service traffic 조회
cm->cm: 배포 schedule 생성
cm->cm: 삭제 schedule 생성
cm->dfs1: schedule 전송
cw1->cw1: schdule file 파싱
cw1->cw1: file 삭제
cw1->dfs2: file 다운로드 요청

loop 모든 schedule이 종료될 떄까지
cw1->dfs: file 다운로드 중 heartbeat file 전송
cm->cm: heartbeat file 확인
opt heartbeat file 이 없을 경우
    cm->cm: heartbeat file 이 없는 worker 의 schedule 종료 처리
end
cw1->cm: file 다운로드 완료 후 response file 전송
cm->cm: response file 확인
opt response file 이 있을 경우
    cm->cm: response file 이 있는 worker 의 schedule 종료 처리
end
cm->cm: 10초 sleep
end
```
