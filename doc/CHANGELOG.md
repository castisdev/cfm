v1.0.0.qr2 / 2020-01-08
===================

[형상변경]
  * 실정 변경
    - 설정 설명, 디폴트값에 대한 셜명은 doc 폴더의 cfm.yml에 있음
    - [storage_usage_limit_percent] -> [remover]-[storage_usage_limit_percent]
    - [task_timeout_sec] -> [taskser]]-[task_timeout_sec]
    - [task_copy_speed_bps] -> [taskser]]-[task_copy_speed_bps]
  * 설정 추가
    - 설정 설명, 디폴트값에 대한 셜명은 doc 폴더의 cfm.yml에 있음
    - [remover]-[remover_sleep_sec]
    - [tasker]-[task_timeout_sec]
    - [servers]-[heartbeat_timeout_sec]
    - [servers]-[heartbeat_sleep_sec]
    - [ignore]-[prefixes]
  * /tasks API response 변경
    - lock 처리 추가
	  - map 형태 -> list 형태로 변경
  * /tasks API 리턴 코드 추가
	  - 500 error 리턴 코드 추가
  * /tasks/{taskId} API 리턴 코드 추가
	  - taskId가 숫자가 아닌 경우 400 error 리턴 코드 추가
    - 없는 id 를 delete 요청 할 경우, 404 error 리턴 코드 추가

[버그]
  * API 응답 시에 header를 중복해서 write 하는 버그 수정
    - https://github.com/golang/go/issues/18761
    - https://github.com/caddyserver/caddy/issues/2537
    - json.NewEncoder(w).Encode(&du) 호출이 성공하면 w.WriteHeader(http.StatusOK)를 호출할 필요없음

[개선]
  * port 충돌로 실행 실패했을 때 화면에 error 메시지 출력하고 종료하게 변경
    - 참고: port  충돌이 없다면 중복실행은 가능 (기존형상과 같음)
  * cfw api df 로 계산되는 used percent 계산방식으로 df 명령어의 계산식을 사용하게 수정함
	  - 이전 버전에서는 df 명령어의  값보다 used percent 값이 작게 나오고 있었음
  * lb event log 에서 hit 수를 만족해서 배포되는 파일(최근 hit수가 증가한 파일)은 삭제 파일 리스트에 포함시키지 않음
	  - 최근 hit가 많이 발생한 파일들이 배포 후에 바로 삭제되는 것을 방지하기 위한 기능
  * 파일 삭제 로그에 파일 정보가 파일 path//파일 이름 식으로 /가 두 번 남는 경우 수정
  * src 서버, dst 서버가 죽었는지 파악하기 위해서 src 서버와 destination 서버에 heartbeat request 기능 추가
  * heartbeat 응답이 확인되지 않은 src 서버, dst 서버는 task 생성 시 포함하지 않는 기능 추가
  * task에 사용 중인 destination 서버는 다른 task에 사용되지 않게 함
	  - src 서버의 갯수가 destination의 갯수보다 많은 경우, 하나의 destination 서버가 여러 task에 사용됨
  * hitcount history file에 두 개 이상의 서버에 위치한 파일을 조사해서 하나의 서버에만 위치하게 관리하는 기능 추가
	  - 하나의 서버 제외한 나머지 서버에 파일 삭제 요청함
  * UpdateStaus 함수 error 조건 완화
	  - 같은 status 에서 같은 status 로 update 할 때 반환값 성공으로 변경
  * task 정보를 파일에도 저장하는 기능 추가
    - leveldb 사용
  * 일부 로그 수정
  * 최신 로그 파일에 대한 symbol link 파일 생성

v1.0.0.qr1 / 2019-11-12
===================
* 최초 릴리즈

[형상변경]
  * task 생성 주기를 5초 -> 1분으로 변경
  * lb eventlog read 시 전체 parsing 하도록 변경

[개선]
  * core 생성 기능 추가

[버그]
  * lb eventlog, grade.info, hitcount.history read 시 빈칸이 있을시 비정상 종료 문제 수정
  * grade.info, hitcount.history read 실패 시 더이상 task 생성하지 않는 문제 수정
