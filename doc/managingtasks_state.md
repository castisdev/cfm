## 구현 내용

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

6. 배포 task를 만들 때 이미 배포 중인 destination 서버 제외

## 요구 사항
1. src -> dest 배포가 쉬지 않고 일어나야 함
2. 하나의 src에서 동시에 여러 개의 배포가 일어나면 안됨
3. 파일 copy 수가 node 당 1로 유지 되어야 함
4. dest 의 disk usage percent 유지
5. 배포하자마자 삭제되는 경우 보완

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


## ASIS:
```plantuml
hide empty description

[*]--> S0
S0 --> S1
S1: DONE, TIMEOUT task 삭제
S1 --> S2
S2 : task 에서 사용 중(배포 중)인 source(server)가 있는 지 체크
S2 --> S3 : 모든 source가 사용 중인경우
S3 : sleep 5초
S3 --> S0

S2 --> S4 : 사용 가능한 source가 있는 경우
S4 : grade 정보, hitcount 정보를 이용하여 배포 파일 목록 생성
note right
 grade 정보에서는 grade 를 구해서 sort 시에 사용
 hitcount 정보에서는 fileSize를 구해서 저장함
 hitcount 는 hitcount 정보에 저장된 순서를 구해서 저장함
 hitcount 정보의 hitcount와 fileSize는 task 만들 때 사용하지 않음
end note

S4 --> S5 : error 가 발생하는 경우
S5 : sleep 5초
S5 --> S0

S4 --> S6 : 정상 처리된 경우
S6 : 광고 파일은 배포 파일 목록에서 제외
S6 --> S7
S7 : 파일 목록을 높은 등급 순서로 정렬

S7 --> S8
S8 : dest의 모든 파일 리스트 수집

S8 --> S9
S9 : LB eventlog 이용하여 파일 목록 생성\n(N분 동안 X번 이상  특정 IP 에서 서비스된 파일 목록)

S9 --> S10
S10 : LB eventlog 이용하여 만든 파일 목록을 큰 hitcount 순서로 정렬

S10 --> S11
S11 : LB eventlog 로 만든 파일 목록과\ngrade 정보, hitcount 정보로 만든 배포 파일 목록 함침

S11 --> ForFileList
ForFileList: 합친 배포 파일 목록으로 배포 task 만들기
note right
 사용 중이 아닌 source 개수만큼
 배포 task가 만들어지면 loop를 빠져나오게 되어있음

 S2 에서 사용 가능한 source가 있는 지 검사하기 때문에,
 여기 꺄지 flow 가 오면 사용가능한 source가 하나라도 있는 상태임
end note

ForFileList --> S11.5
S11.5 : 다음 파일 선택 (순서대로 선택)

S11.5 --> S12
S12 : 배포 중인 파일인지 검사
note right
 source와 dest를 확인하진 않음
end note
S12 --> ForFileList : 배포 중인 파일인 경우

S12 --> S13 : 배포 중인 파일이 아닌 경우
S13 : dest 에 있는 파일인지 검사
S13 --> ForFileList : dest에 있는 파일인 경우

S13 --> S14 : dest에 없는 파일인 경우
S14 : source path 에 없는 파일인지 검사
S14 --> ForFileList : source path 에 없는 파일인 경우

S14 --> S15 : source path 에 있는 파일인 경우
S15 : 사용 가능한 source가 있는 지 검사

S15 --> S17: 사용 가능한 source가 있는 경우
S17 : source 선택

S17 --> S18
S18: 다음 dest 선택 (ring 구조체에 저장되어있음)
note right
 dest가 하나도 없는 경우에 대한 처리는 없음
end note

S18 --> S19
S19: 선택한 {source, dest, filepath, filename, etc.}의 task 생성

S15 --> S16: 사용 가능한 source가 없는 경우
S16 : sleep 60
S16 --> S0

S19 --> S16 : 더 이상 배포할 피일이 없는 경우
S19 --> ForFileList : 배포할 파일이 남은 경우
```
