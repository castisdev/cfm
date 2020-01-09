# API

## GET /tasks
- task 목록 조회
- Response:
  - 200 OK
  - 500 Internal Server Error
```json
 [
          {
            "id": "1578376668044673074",
            "ctime": 1578376668,
            "mtime": 1578376668,
            "status": "done",
            "src_ip": "127.0.0.1",
            "dst_ip": "",
            "file_path": "/data2/A.mpg",
            "file_name": "A.mpg",
            "grade": 0,
            "copy_speed": "",
            "src_addr": "127.0.0.1:8080",
            "dst_addr": "127.0.0.1:8081"
          },
          {
            "id": "1578376668154733282",
            "ctime": 1578376668,
            "mtime": 1578376668,
            "status": "ready",
            "src_ip": "127.0.0.1",
            "dst_ip": "",
            "file_path": "/data2/C.mpg",
            "file_name": "C.mpg",
            "grade": 0,
            "copy_speed": "",
            "src_addr": "127.0.0.1:8080",
            "dst_addr": "127.0.0.2:8081"
          },
          {
            "id": "1578376668154746815",
            "ctime": 1578376668,
            "mtime": 1578376668,
            "status": "working",
            "src_ip": "127.0.0.1",
            "dst_ip": "",
            "file_path": "/data2/D.mpg",
            "file_name": "D.mpg",
            "grade": 0,
            "copy_speed": "",
            "src_addr": "127.0.0.1:8080",
            "dst_addr": "127.0.0.3:8081"
          },
          {
            "id": "1578376668154755177",
            "ctime": 1578376668,
            "mtime": 1578376668,
            "status": "timeout",
            "src_ip": "127.0.0.1",
            "dst_ip": "",
            "file_path": "/data2/E.mpg",
            "file_name": "E.mpg",
            "grade": 0,
            "copy_speed": "",
            "src_addr": "127.0.0.1:8080",
            "dst_addr": "127.0.0.4:8081"
          }
        ]
```
- task : cfm이 만든 파일 배포(전송) 작업
  - src, dest , file 이름, 상태 등의 속성 값이 있음
- task 속성 값
  - id : task 의 ID, unique key 값
  - ctime : create 시간, unix time
  - mtime : modified 시간, unix time
  - status : 상태값
    - ready : 배포 준비 상태
    - working : src에서 dest로 file이 배포 중인 상태
    - done : src에서 dest로 file 이 배포 완료된 상태
    - timeout : src에서 dest로 file 이 배포 timeout된 상태
  - src_ip : src 서버의 ip
  - dest_ip : dest 서버의 ip
  - file_path : 파일의 폴더 위치/파일 이름
  - file_name : 파일 이름
  - grade : 파일의 등급, 일반적으로 등급 값이 작을 수록 배포 우선순위가 높음
  - copy_speed : 파일 배포 시 참고하는 속도
  - src_addr : src 서버의 ip, port 값
  - dest_addr : dest 서버의 ip, port 값


- curl 사용 예:
```bash
    $ curl 127.0.0.1:7888/tasks
```
- httpie 사용 예:
```bash
    $ http 127.0.0.1:7888/tasks
```

## DELETE /tasks/{taskId}
- task 삭제
- Response:
  - 200 OK
  - 400 Bad Request
  - 404 Not Found

- curl 사용 예:
```bash
  $ curl -X DELETE 127.0.0.1:7888/tasks/1578383370370052104
```
- httpie 사용 예:
```bash
  $ http DELETE 127.0.0.1:7888/tasks/1578383370370052104
```

## PATCH /tasks/{taskId}
- task status 변경
- Request:
```json
{"status":"done"}
```
- Reponse:
  - 200 OK
  - 400 Bad Request
  - 404 Not Found

- curl 사용 예:
```bash
$ curl --header "Content-Type: application/json" \
  --request PATCH \
  --data '{"status":"done"}' \
  http://127.0.0.1:7888/tasks/1578383370370052104
```
- httpie 사용 예:
```bash
$ http PATCH 127.0.0.1:7888/tasks/1578383370370052104 \
  status=done --verbose
```
