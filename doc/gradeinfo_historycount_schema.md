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
