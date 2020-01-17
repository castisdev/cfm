2019-12-12 작성

## GRADEINFO
첫번째 line은 header line : column 이름 목록
칼럼 구분자 : tab
예) filename  weightcount bitrate grade
| Column 이름     | 설명                        |
|-----------------|-----------------------------|
| filename        | 파일 이름 (unique)|
|	weightcount     | hitcount 와 bitrate의 가중치로 계산하도록 (정렬key) |
| bitrate         | 파일 bitrate |
| grade           | 등급 |
| sumHitCount     | 이번 조사시 hitocunt |
| historyCount    | 몇 개의 history count로 되어있는지 |
| TargetCopyCount<br>(TargetDistributeWeight) | 등급에 따른 필요한 배포 가중치 |

## HITCOUNTHISTORY
첫번째 line은 header line : history 번호인듯
예) historyheader:1524468282

칼럼 구분자 : ,(콤마)
마지막 칼럼 구분자는 : =
마지막 5개의 칼럼(basecount, sharedcount, virtualVODcount, contentType, hitcountHistory) 정보는 부정확함
| Column 이름       | 설명                        |
|-------------------|-----------------------------|
| mediafilename     | 파일 이름 (unique)|
|	registertime      | 파일의 마지막 write시간 |
| bitrate           | 파일 bitrate |
| mediafilesize     | 파일 크기(byte) |
| vodDistributelist | 미디어 파일을 보유하고 있는 vod 리스트(공백 구분) |
| basecount         | base 존재 여부   |
| sharedcount       | aux 존재 여부   |
| virtualVODcount   | virtualVOD 존재여부  |
| contentType       | 타입번호= (0=,1=,2=,3=,4=,5=) |
| hitcountlist   | hitcount 리스트(공백 구분) |

## RISING HITCOUNT 구하기
Event log(lb 또는 glb의 log 로 추정)의 로그 중에
아래 형식의 로그를 찾아
특정 시간대, 특정 IP로 요청이 온 파일 이름별 count 를 구한다.

**0x40ffff**,1,**1527951607**,Server **125.159.40.3** Selected for Client StreamID : 609d8714-096a-475e-994c-135deea7177f, ClientID : 0, GLB IP : 125.159.40.5's **file(MZ3I5008SGL1500001_K20180602222428.mpg)** Request

- 첫번째 필드는 0x40ffff 로 시작하고,
- 세번째 필드인 로그 시간이 기준 시간과 같거나 크고,
- 네번째 필드에 특정 IP가 있고, file(파일이름)로 되어있는 문자열이 있는 경우를 구한다.
- 필드 구분자는 ,이고
- 파일이름은 숫자와 영문문자와 .과 -와 _로 이루어진 문자열로 구성된다. (한글 이름 등은 파싱 안됨)
