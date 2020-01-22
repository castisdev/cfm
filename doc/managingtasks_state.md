## 요구 사항
1. src -> dest 배포가 쉬지 않고 일어나야 함
2. 하나의 src에서 동시에 여러 개의 배포가 일어나면 안됨
3. 파일 copy 수가 node 당 1로 유지 되어야 함
4. dest 의 disk usage percent 유지
5. 배포하자마자 삭제되는 경우 보완

## 구현 내용

1. dest cfw가 급 죽는 경우, 1번 요구사항 불만족:
  죽은 dest cfw 에 대한 배포 task 를 만들면,
  해당 dest cfw가 배포 task 를 수행하지 못하므로,
  해당 배포 task는 timeout 이 날 때, 없어지지 않고, 배포가 일어나지 않음
  - heartbeat check 구현하여, heartbeat가 없는 dest cfw에 대한 배포 task는 만들지 않도록 구현

2. dest cfw가 급 죽는 경우, 3번 요구사항 불만족:
  죽은 cfw로 부터 파일 리스트를 가져올 수 없기 때문에,
  죽은 cfw에 존재하는 파일에 대해서 배포 task를 만들 수 있음
  나중에 죽은 cfw가 살아나면, 하나의 파일의 여러 dest 에 존재할 수 있음
  - copy수 1개를 유지하기 위해서
    - copy 수가 여러 개인 파일에 대해서 cfw에게 삭제 명령을 날리는 기능을 구현
    - 배포 시에 이미 서버에 있는 파일에 대한 배포 task는 만들지 않음
      - 기존에 있던 기능에 더해서 hitcount history file에 있는 서버 위치 정보도 이용

3. 5번의 배포되지마자 삭제되는 case 가 발생
  grade는 낮지만, hit count가 급 올라간 파일이 배포 대상이 되고,
  grade가 낮기 때문에, 동시에 삭제 대상이 되는 현상이 생길 수 있다고 함
  - 급 hit count가 올라간 파일에 대해서는 삭제에서 제외
  - 급 hit count가 올라간 파일도 copy 수가 하나 이상이 되는 경우에는 삭제

4. hit count 가 급 올라가는 파일이 광고 파일인 경우
  - 광고 파일인 경우 모든 삭제 및 배포에서 제외
    - 즉, copy 수가 하나 이상이되는 경우에도 삭제 제외

5. delete 명령 내릴 때, log 보완
  로그에 파일 경로가 남을 때 파일이름에 / 가 두 번 들어가는 현상 수정

6. 배포 task를 만들 때 이미 배포 중인 destination 서버 제외
  - 배포 중인 dest 서버에 대해서는 배포 task를 만들지 않게 함
    - 배포 가능한 src 서버가 여러 개이고, dest 서버는 하나 밖에 없는 경우엔
     src 서버 갯수만큼 배포 task를 만듬

OK CASE1
``` plantuml
left to right direction
rectangle src {
  (src1)
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src2) --> (dest2)
(src1) --> (dest1)
```

OK CASE2
``` plantuml
left to right direction

rectangle src {
  (src1)
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) --> (dest1)
(src2) --> (dest1)
```

NG CASE1
``` plantuml
left to right direction
rectangle src {
  (src1)
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) --> (dest1)
(src1) --> (dest1)
```

NG CASE2
``` plantuml
left to right direction
rectangle src {
  (src1)
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) --> (dest1)
(src1) --> (dest2)
```



## 문제점
1. LB eventlog를 이용하여 만든 파일 목록에서 광고 파일 검사하는 부분이 없음
2. source에 장애가 나는 경우, 해당 source 를 이용해서 만들어진 배포 task는 TIMEOUT 날 때까지 정리되지 않는다고 함
  코드 상으로는 이 경우에, source만 장애가 나는 경우, dest 가 해당 task 의 상태를 DONE을 만들 것으로 보임
  -> 정리가 안되던 test 환경은 source 와 dest가 같은 서버인 경우였을 수도 있다고 함

``` plantuml
left to right direction
skinparam usecase {
BackgroundColor<<error>> red
BorderColor<<error>> darkred
}
rectangle src {
  (src1)<<error>>
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) --> (dest1)
(src2) --> (dest1)
```

``` plantuml
left to right direction
skinparam usecase {
BackgroundColor<<error>> red
BorderColor<<error>> darkred
}
rectangle src {
  (src1)<<error>>
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) .- (dest1) : dest1이 DONE 보고를 해서,\n삭제되지만\n src1 의 error 상태를 모르기 때문에 다시 task 가 만들어짐
(src2) --> (dest1)
```

``` plantuml
left to right direction
skinparam usecase {
BackgroundColor<<error>> red
BorderColor<<error>> darkred
}
rectangle src {
  (src1)<<error>>
  (src2)
}
rectangle dest {
  (dest1)
  (dest2)
  (dest3)
}
(src1) --> (dest1) : 만들어지고 삭제되는 일이 반복됨
(src2) --> (dest1)

```

3. dest가 장애 나는 경우, 해당 배포 task의 src 를 사용할 수가 없는 상황이 발생함

아래 상황에서 dest1 에 장애가 발생하면,
cfw 가 배포 상태를 바꾸어줄 수 없기 때문에,
cfm 에서 timeout 처리할 때까지 새로운 배포 task 가 만들어지지 않음

``` plantuml
left to right direction
skinparam usecase {
BackgroundColor<<error>> red
BorderColor<<error>> darkred
}

rectangle src {
  (src1)
  (src2)
}
rectangle dest {
  (dest1)<<error>>
  (dest2)
  (dest3)
}
(src1) --> (dest1)
(src2) --> (dest1)
```

4. 배포되지마자 삭제되는 경우 보완

코드 상으로 볼 때,
source 에 장애가 나는 경우,
해당 배포 task를 정리할 수 있는 방법이
배포 task 가 DONE이 되거나
TIMEOUT 처리되는 경우임
해당 배포 task 가 정리 된 후에
장애 상태의 source를 가지고 배포 task를 다시 만들 게되면
같은 현상이 계속 반복될 것으로 예상됨

cfw 코드를 볼 때, download 시에 장애가 나도 배포 상태를 DONE으로 만들기 때문에
TIMEOUT 이전에 정리될 것으로 예상되는데,
왜 TIMEOUT 때까지 정리 안된다고 한것인지 더 알아볼 필요가 있음

## 개선 방안
1. LB eventlog를 이용하여 만든 파일 목록에 대해서 광고 파일 검사하는 기능 추가
2. source에 장애 상태를 추가하여, 배포 task 만들 때 해당 source는 제외하기
3. task에 장애 상태 추가하여, 해당 task 에 사용된 source, dest 를 이용한 task 가 만들어지는 경우 막기


## v1.0.0.qr2 / 2019-11-12

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

## v1.0.0.qr1 / 2019-11-12

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
