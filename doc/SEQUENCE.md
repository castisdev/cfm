v1.0.0.qr2 / 2020-01-08
===================
cfm, cfw, DFS

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


v1.0.0.qr1 / 2019-11-12
===================
cfm, cfw, DFS

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
