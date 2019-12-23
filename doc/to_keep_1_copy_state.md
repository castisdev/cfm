# TO KEEP 1 COPY IN DESINATION SEVERS
하나의 파일이 여러 destination에 있는 경우 하나의 파일만 유지하기

1. reference가 되는 hicount.hisotry의 정보를 이용하여 작업하기

hicount.hisotry 에는 파일과 파일이 위치한 서버 ip list 가 있으므로
서버 ip list 에 서버 ip가 두 개 이상일 때,
서버 ip list 에 서버 ip가 하나만 남을 때까지
서버에 해당 file 삭제 요청을 날린다.

조건:
1. 서버가 중간에 추가, 삭제되지 않아야 한다.
2. 서버 ip 순으로 sort 해서 삭제 요청 순서를 항상 같게 유지해야 한다.
3. hitcount.history를 한 번 읽고, 파일 별로 삭제 요청은 1회 번씩만 한다.
	또는 hitcount.history를 한 번 읽고, 파일 별로 서버 ip가 n개 있을 때, 삭제 요청은 n-1 회 한다.


CASE1:
----

``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 1. read hitcount.history file
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note

(cfm) --> (dest1) : 2. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 3. 요청 성공

```

``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest2 : a.mpg
dest3 : a.mpg
end note

(cfm) --> (dest2) : 5. a.mpg 가 dest2, dest3 에 있는 것을 발견하고,\ndest2 에 삭제 요청 a.mpg
(dest2) --> (cfm) : 6. 요청 성공
```

``` plantuml
left to right direction
(dest3) --> (a.mpg)

(hitcount.history) <-- (cfm) : 7. read hitcount.history file again
note top
dest3 : a.mpg
end note

(cfm) --> (cfm): 8. a.mpg 가 dest3 에만 있는 것을 확인하고 아무 일도 하지 않음
```

``` plantuml
left to right direction
(dest3) --> (a.mpg)
note right
dest3 에만 a.mpg 유지
end note
```

CASE2:
----
hitcount.history 파일 update 와 cfm의 read가 sync 가 맞지 않을 경우,
파일 삭제 요청 순서가 항상 같은 경우,
hitcount.history 파일이 update되기 까지
계속해서 요청 실패하고,
update 된 후에, CASE1 처럼 되지만,
삭제 요청 순서가 다른 경우, 문제 발생함

``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)


(hitcount.history) <-- (cfm) : 1. read hitcount.history file
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest1) : 2. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 3. 요청 성공
```

``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest1) : 5. 여전히 a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(cfm) <-- (dest1) : 6. 요청 실패
```

hitcount.history file이 update 되기 전에는 dest1에 요청 실패하다가
hitcount.history 가 update 되면 dest2에 요청
``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest2) : 5. a.mpg 가 dest2, dest3 에 있는 것을 발견하고,\ndest2 에 삭제 요청 a.mpg
(cfm) <-- (dest2) : 6. 요청 성공

```

``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest2) : 5. 여전히 a.mpg 가 dest2, dest3에 있는 것을 발견하고,\ndest2 에 삭제 요청 a.mpg
(cfm) <-- (dest2) : 6. 요청 실패
```
hitcount.history file이 update 되기 전에는 dest2에 요청 실패하다가
hitcount.history 가 update 되면 dest3에만 있는 것으로 확인하고 멈춤

``` plantuml
left to right direction
(dest3) --> (a.mpg)

(hitcount.history) <-- (cfm) : 7. read hitcount.history file again
note top
dest3 : a.mpg
end note

(cfm) --> (cfm): 8. a.mpg 가 dest3 에만 있는 것을 확인하고 아무 일도 하지 않음
```

``` plantuml
left to right direction
(dest3) --> (a.mpg)
note right
dest3 에만 a.mpg 유지
end note
```

CASE3:
----
hitcount.history 파일 update 와 cfm의 read가 sync 가 맞지 않을 경우,
파일 삭제 요청 순서가 다른 경우,
모든 파일이 삭제될 수 있음

``` plantuml
left to right direction
(dest3)-->(a.mpg)
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)


(hitcount.history) <-- (cfm) : 1. read hitcount.history file
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest1) : 2. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 3. 요청 성공
```
삭제 요청 순서가 달라져서 dest1이 아닌, dest2 에 요청하면 요청 성공하고,
이어서 dest3 의 파일까지 삭제 요청이 성공하게됨
``` plantuml
left to right direction
(dest3)--> (a.mpg)
(dest2)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest2) : 5. 여전히 a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest2에 삭제 요청 a.mpg
(cfm) <-- (dest2) : 6. 요청 성공
```

``` plantuml
left to right direction
(dest3)--> (a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file again
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest3) : 5. 여전히 a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest3에 삭제 요청 a.mpg
(cfm) <-- (dest3) : 6. 요청 성공
```

모든 destination 의 파일이 지워질 수 있음

**CASE2 보완 방안 :**
hitcount.history이 update된 파일인지 확인하기
효과 : 요청 실패 횟수를 줄일 수 있음

**CASE3 보완 방안 :**
삭제 요청 순서를 동일하게 유지 하기
효과 : 동일하게 유지 하지 않으면 모든 destination의 파일이 삭제될 수 있음

CASE4:
----
hitcount.history file이 update됨에도 불구하고
destination과 통신이 안되는 등의 이유로
destination의 최신 정보가 반영이 안되지만,
파일 수가 증가하지 않고 감소만 하는 경우

CASE2, CASE3과 같은 경우여서 결국 파일 수를 하나로 유지할 수 있음

CASE4 보완 방안 :
CASE3 과 같음, 삭제 요청 순서가 계속 같아야 함


``` plantuml
left to right direction
(dest3)-->(a.mpg)
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)


(hitcount.history) <-- (cfm) : 1. read hitcount.history file **V1**
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest1) : 2. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 3. 요청 성공
```

``` plantuml
left to right direction
(dest3)-->(a.mpg)
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)


(hitcount.history) <-- (cfm) : 4. read hitcount.history file **V2**
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg
end note
(cfm) --> (dest1) : 5. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 6. 요청 실패
```

CASE5:
----
hitcount.history file이 update되고
파일 삭제 순서는 일정하게 유지했으나,

destination과 통신이 안되는 등의 이유로
destination의 최신 정보가 반영이 안되어
파일 수가 증가하는 경우

삭제요청 순서가 일정하면, 파일 copy 수가 1로 유지됨

``` plantuml
left to right direction
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)
(dest3)-->(a.mpg)

(hitcount.history) <-- (cfm) : 1. read hitcount.history file
note top
dest2 : a.mpg
dest3 : a.mpg

dest1 에 a.mpg가 있는 것이 반영이 안된 경우
end note
(cfm) --> (dest2) : 2. a.mpg 가 dest2, dest3 에 있는 것을 발견하고,\ndest2 에 삭제 요청 a.mpg
(dest2) --> (cfm) : 3. 요청 성공
```

``` plantuml
left to right direction
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)
(dest3)-->(a.mpg)

(hitcount.history) <-- (cfm) : 4. read hitcount.history file
note top
dest1 : a.mpg
dest2 : a.mpg
dest3 : a.mpg

dest1 는 반영이 되고,
dest2 는 반영이 안되어, 파일 수가 증가한 경우
end note
(cfm) --> (dest1) : 5. a.mpg 가 dest1, dest2, dest3 에 있는 것을 발견하고,\ndest1 에 삭제 요청 a.mpg
(dest1) --> (cfm) : 6. 요청 성공
```

dest1 삭제 요청 후에
dest1 반영이 안되면, 계속 dest1 에 삭제 요청을 하고,
dest1 반영이 되면, dest2에 삭제 요청을 한다.
dest2 는 이전에 삭제 요청 성공했으므로, 요청 실패가 일어난다.

``` plantuml
left to right direction
(dest1)-->(a.mpg)
(dest2)-->(a.mpg)
(dest3)-->(a.mpg)

(hitcount.history) <-- (cfm) : 1. read hitcount.history file
note top
dest2 : a.mpg
dest3 : a.mpg

dest1 에 a.mpg가 삭제된 것이 반영 된 경우
end note
(cfm) --> (dest2) : 2. a.mpg 가 dest2, dest3 에 있는 것을 발견하고,\ndest2 에 삭제 요청 a.mpg
(dest2) --> (cfm) : 3. 요청 실패
```
