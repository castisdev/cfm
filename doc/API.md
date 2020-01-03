# API

## GET /api/caches/{host}/{filePath:.*}
- 캐시 import url 목록 조회
- Response
```json
[
    "http://1.1.1.1:8181/api/caches/httpfsrv/a.mp4",
    "http://1.1.1.2:8181/api/caches/httpfsrv/a.mp4",
    "http://1.1.1.3:8181/api/caches/httpfsrv/a.mp4"
]
```
- 전체 streamer 목록 중 우선 순위에 따라서 import url 목록 응답함

## DELETE /api/caches/{host}/{filePath:.*}
- 캐시 purge
- Response:
  - 204 No Content

## POST /api/caches/{host}/{filePath:.*}
- 캐시 import
- multipart/form-data 로 파일 업로드
  - form-data 의 name 필드에 "uploadfile" 값을 써주어야 함
  - form-data 의 filename 필드는 사용하지 않고 api url 의 {filePath:.*} 파트를 파일 이름으로 사용함
- Request:
```
POST /api/caches/172.16.45.13:8082/sample.mpg HTTP/1.1
Content-Type: multipart/form-data; boundary=a8e358c39ead463290ef70f4f5b6f024
Content-Length: 5620976

--a8e358c39ead463290ef70f4f5b6f024
Content-Disposition: form-data; name="uploadfile"; filename="sample.mpg"
[BINARY DATA]
```
- Response:
  - 201 Created

- CURL 사용 예
```bash
$ curl -F "uploadfile=@~/Downloads/sample.mpg" localhost:8092/api/caches/172.16.45.13:8082/sample.mpg
```

- HTTPie 사용 예
```bash
$ http -f POST localhost:8092/api/caches/172.16.45.13:8082/sample.mpg uploadfile@~/Downloads/sample.mpg
```
