## 2020-01-21
- 메모리 누수 방지 위한 코드 추가, keep alive 설정 시에 문제 방지 코드 추가
  - // https://stackoverflow.com/questions/33229860/go-http-requests-json-reusing-connections
   - // https: //stackoverflow.com/questions/17959732/why-is-go-https-client-not-reusing-connections

## 2020-01-16
- FIXME: disk 상태도 고려하면 좋을 듯
- 삭제할 때는 전체 file meta 에도 있고, 서버에도 있어야 함
- 배포할 때는 전체 file meta 에는 있고, 서버에는 없어야 함
  변경 :
    급상승 hits 파일은 meta 가 없어도 배포가 됨
    -> meta가 없으면 배포 안되게 함
      -> meta가 없으면 지울 수도 없기 때문에, 배포도 안하는 게 좋을 듯

- gradeinfo, hitcount.history 로 구한 전체 파일 meta list에 없는 파일에 대한 삭제, 배포가 가능한가?
  : 현재 상태
    : 삭제 안됨
      - file meta에서 size, 위치 서버 정보를 구할 수 없어서 삭제 불가
      - test 코드에 추가하기
    : 배포 안됨, 단) 급상승 hits 파일은 meta 가 없어도 배포가 됨
      - log(lb log?) parsing으로 구한 급 hit 상승 파일이 file meta에는 없는 경우
        - 파일 이름만 알면 배포 가능하지만 일단 제외시킴
      - test 코드에 추가하기

- 중복된 파일의 삭제 요청의 경우, 요청하지 말아야하는 파일에 대한 요구사항이 있는가?
 : disk 용량 모자란 경우 삭제에서 제외되는 파일의 경우와 같은 예외 사항 추가
  * SAN에 없는 경우
  * 특정 prefix 로 시작하는 경우
 : disk 용량 모자란 경우와 다른 점
  * hitocunt 가 급히 상승한 경우는 지움

## 2020-01-14
- 중복된 파일의 삭제 요청의 경우, 요청하지 말아야하는 파일에 대한 요구사항이 있는가?
  : disk 용량이 모자란 경우 삭제에서 제외되는 파일
    * SAN에 없는 경우
      : **SAN에 있는 파일만 FilePath를 알아내어 task를 만들 수 있음**
    * 특정 prefix 로 시작하는 경우
      : **광고 파일 같은 경우 제외**
    * hitcount 가 급히 상승한 경우
  : 현재는 제외 되는 경우 없음

- 중복 위치 파일이어서 삭제 요청한 파일에 대해서
나중에 disk 부족으로 다시 삭제 요청하는 지 test 코드 만들기, 리뷰 하기
  : test 코드 만들고, 코드 보완함
- 여러 dest 에 중복되어 있는 파일에 대해서만 삭제 요청하는 지 test 코드 만들기, 리뷰 하기
  : test 코드 만듬

## 2020-01-08
- 중복 위치 파일이어서 삭제 요청한 파일에 대해서
나중에 disk 부족으로 다시 삭제 요청하는 지 test 코드 만들기, 리뷰 하기
- 여러 dest 에 중복되어 있는 파일에 대해서만 삭제 요청하는 지 test 코드 만들기, 리뷰 하기

## 2020-01-03
이미 cfm 이 실행 중인데, 또 실행 중일 때
메시지 없이 그냥 내려가는 것 같음

## 2019-12-23
Q: dest disk 용량이 모자라서, 파일 remove 할 때, 광고 파일은 제외하게 되어있는데,
원래 요구사항이었다고 생각하고 그대로 두었음

## 2019-12-12

tasks.UpdateStatus 함수의 error 조건 완화
같은 status 에서 같은 status 로 update할 때,
실패 -> 성공 으로 변경

loglevel 변경 : timeout 으로 지워지는 task 에 대한 log level을 warning에서 info 로 변경

config 추가
listen_addr

~~task 의 srcIP 와 destIP 를 srcAddr(IP:port), destAddr(IP:port) 로 바꿈~~
task 에 srcAddr(IP:port), destAddr(IP:port) 추가
task 의 srcIP 와 destIP 는 삭제 예정

## 2019-12-11

- net/http 의 keep-alive 속성은?
- func (tasks *Tasks) UpdateStatus(id int64, s Status) error
  - 에러 처리 변경 예정
  - 같은 status 로 변경 시 error 발생 기능 제거 예정

### 요구 사항
1. src -> dest 배포가 쉬지 않고 일어나야 함
2. 하나의 src에서 동시에 여러 개의 배포가 일어나면 안됨
3. 파일 copy 수가 node 당 1로 유지 되어야 함
4. dest 의 disk usage percent 유지
5. 배포하자마자 삭제되는 경우 보완

### 구현 내용

1. dest cfw가 급 죽는 경우, 1번 요구사항 불만족
  heartbeat check 구현을 해야할 듯 하네요.

2. dest cfw 가 급 죽는 경우, 3번 요구사항 불만족
  copy 수 1개를 유지하기 위해서 삭제 명령을 날리는 기능이 필요한 것 같습니다.

3. 배포되지마자 삭제되는 case 가 발생
  grade는 낮지만, hit count가 급 올라간 파일이 배포 대상이 되고,
  grade 가 낮기 때문에, 동시에 삭제 대상이 되는 현상이 생길 수 있다고 합니다.
    급 배포 대상 파일에 대해서는 grade를 올리는 기능이 필요할 것 같습니다.

4. hit count 가 급 올라가는 파일이 광고 파일인 경우
  배포 대상에서 제외하는 코드가 없는 것 같아서 추가해야 할 것 같습니다.

5. delete 명령 내릴 때, log 보완
  파일이름에 / 가 두 번 들어간다고 함

## 2019-12-09

문제점 및 개선 의견:

장애가 있는 dest server 가 있을 경우,
처리 안되는 schedule 이 쌓여서,
배포 schedule이 더 이상 만들어지지 않는다고 함

한 시간에 이상된 schedule 이 지워지는데,
한 시간이라는 설정을 줄이는 방법 대신,
장애가 있는 dest server를 schedule에서 지우는 기능이 있었으면
좋겠다고 함


### 코드 리뷰:

#### tasker.task 의 함수

FindTaskByID 함수 : 그냥 map 함수 사용하면 될 듯

#### tasker.RunForever() 함수

- 3번 comment 가 없음
  SrcServers 의 개수와 selected 개수가 같다는 게 어떤 의미인지 파악 필요
  src is full 이라는 것이 무슨 뜻인가?
  이 경우에, 5초간 sleep 됨 (설정 없음)

해석:
task queue 에 있으면 source 가 할당된 상태로 변경
할당된 source 개수가 원래 source 개수와 같을 때
 ? source 가 같고, dest 나 파일이름이 다른 task 가 여럿인 경우에도
 n 이 증가할 수 있을 것 같은데...
 이 경우에도 src full 상황이 생길 듯

**src is full 상황이 어떤 의미인지?**
  모든 src가 selected 상태인 경우 check 라면
  task 에 src 가 모두 다르다는 가정이 필요할 것 같음
  -> 뒤에 코드를 보면, task 가 source 개수만큼만 만들어짐
  ->  아래 코드는 task 가 source 개수만큼 만들어져있으면,
  5초 sleep 한다는 의미가 되는 것 같음

문제의 코드
``` go
		if n == len(*SrcServers) {
			cilog.Debugf("src is full")
			time.Sleep(time.Second * 5)
			continue
		}
```

문제의 코드가 속한 주변 코드
``` go

for {

		CleanTask(tasks)

		// 1. unset selected flag (true->false)
		for _, src := range *SrcServers {
			src.selected = false
		}

		// 2. task queue 에 있는 src ip는 할당된 상태로 변경
		n := 0
		for _, task := range tasks.TaskMap {
			for _, src := range *SrcServers {
				if src.IP == task.SrcIP {
					src.selected = true
					n++
				}
			}
		}

		if n == len(*SrcServers) {
			cilog.Debugf("src is full")
			time.Sleep(time.Second * 5)
			continue
		}

		// 4. 파일 등급 list 생성
		fileMetaMap := make(map[string]*common.FileMeta)


```

- // 4. 파일 등급 list 생성
grade 파일 parsing 실패 시에 5초 sleep : ?
hitcount 파일 parsing 실패 시에 5초 sleep : ?


- map 에 넣고, slice 로 바꾼 후 sort 하는 것과
 heap 을 사용하는 것의 속도 차이가 있을지?

- CollectRemoteFileList 함수에서
  코드 test 하기 :
  remoteFileSet map 에 file 이라는 key 가 없을 때
    remoteFileSet 초기화는 어떻게 되는 건가?
    아래와 같은 코드는 어떻게 동작할까?
    remoteFileSet[file]++
      테스트 해보니 : 없은 file 에 대해서 수행하면, 초기값 0 으로 처리되어 1로 처리됨
  hitcount 는 int 로 충분한가? -> int64 라면 충분
  error 가 나면, 결과 file list 에 아무것도 안들어있을 수도 있을 듯

- tasker_test 에 RunForever test 코드가 없음
  몇가지 case를 simulation 해서 test 코드를 만들어보면 어떨지
  실제 코드를 고치지 않고 test code 만드는 법을 몰라서
  (mockup 만드는 법 등을 몰라서)
  코드를 직접 고치든지 다른 program으로 만들어서??? test 해보아야 할 듯

- // 9. LB EventLog 에서 특정 IP 에 할당된 파일 목록 추출
hitcount.history 정보보다,
lblog 에서 구한 hitcount 값을 우선으로 함


- lb log 에서 구한 flie list 와
hitcount 에서 구한 file list 에 중복되는 file 이 있을 수도 있을 것 같은데
고려가 되어있지는 않는 것 같음
  - 중복이 없는 것인지?
  >> 중복이 있겠지만, 나중에 task 만들 때, 중복되는 파일을 배포하지 않음

- lb log 에서 구한 file list 에서는 광고 file 제거하는 기능이 없음
  >> 추가 필요할 듯
- // 12. 이미 task queue 에 있는 파일이면 ignored
task 의 key 가 file name이 아닌데 왜 ignored 할 까?
 >> file 을 1 copy 유지하는 정책이 있어서, 여러 destinatioh 에 배포할 필요가 없기 때문

``` go
// 12. 이미 task queue 에 있는 파일이면 ignored
if _, exists := tasks.FindTaskByFileName(file.Name); exists {
  // cilog.Debugf("%s is already in task queue", file.Name)
  continue
}
```

- task 만드는 loop
  - source 선택 :
    selected 되지 않는 source 를 선택
    - source 가 없을 때,
      src is full 이라는 로그가 남으면서 task 만드는 loop 종료됨
      - 앞에서는 selecte 할 것이 없으면 sleep 했었는데?
    - 하나의 src 는 여러 task 에 들어갈 수 없나 봄?

  source 개수만큼만 task 를 만들고, 끝나고,
  60초 sleep 하고 나서
  다시 file list 구하는 작업을 다시 하는데,
  결국 60초 한 번 씩 source 개수만큼만 task 가 만들어지는 것 같음


- task 만드는 loop 가 끝나면 60초 sleep

- 특정 dest server 에만 file 이 없는 경우 처리가 어떻게 되는지 파악 안됨
>> dest server 중 하나에만 file을 배포하면 됨
