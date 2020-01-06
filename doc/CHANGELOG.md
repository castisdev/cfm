v1.0.0.qr2 / 2020-01-XX
===================
[형상변경]
  * 실정 변경
    * 설정 설명, 디폴트값에 대한 셜명은 doc 폴더의 cfm.yml에 있음
    * [storage_usage_limit_percent] -> [remover]-[storage_usage_limit_percent]
    * [task_timeout_sec] -> [taskser]]-[task_timeout_sec]
    * [task_copy_speed_bps] -> [taskser]]-[task_copy_speed_bps]
  * 설정 추가
    * 설정 설명, 디폴트값에 대한 셜명은 doc 폴더의 cfm.yml에 있음
    * [remover]-[remover_sleep_sec]
    * [tasker]-[task_timeout_sec]
    * [servers]-[heartbeat_timeout_sec]
    * [servers]-[heartbeat_sleep_sec]
[개선]
  * port 충돌로 실행 실패했을 때 화면에 error 메시지 출력하고 종료하게 변경
    * 참고: port  충돌이 없다면 중복실행은 가능 (기존형상과 같음)

v1.0.0.qr1 / 2019-11-12
===================
* 최초 릴리즈
