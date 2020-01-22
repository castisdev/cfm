v1.0.0.qr2 / 2020-01-23
===================

### cfm, cfw, DFS

- 배포 schedule과 task는 같은 의미로 사용됨
- cfw 와 heartbeat 추가
- 여러 서버에 배포된 파일을 한 서버에만 남기고 나머지에서는 제거하는 기능 추가

![cfm_cfw_DFS_Sequence_v1.0.0.qr2.png](cfm_cfw_DFS_Sequence_v1.0.0.qr2.png)

<br><br>

### tasker
- task 관리 모듈
![cfm_v1.0.0.qr2_tasker_State.png](cfm_v1.0.0.qr2_tasker_State.png)

![cfm_v1.0.0.qr2_tasker_make_task_files_State.png](cfm_v1.0.0.qr2_tasker_make_task_files_State.png)

### remover

- 파일 삭제 요청 모듈

![cfm_v1.0.0.qr2_remover_State.png](cfm_v1.0.0.qr2_remover_State.png)

![cfm_v1.0.0.qr2_remover_delete_duplicated_files_State.png](cfm_v1.0.0.qr2_remover_delete_duplicated_files_State.png)

![cfm_v1.0.0.qr2_remover_delete_files_for_disk_free_State.png](cfm_v1.0.0.qr2_remover_delete_files_for_disk_free_State.png)

<br><br>
v1.0.0.qr1 / 2019-11-12
===================

### cfm, cfw, DFS

- 배포 schedule과 task는 같은 의미로 사용됨

![cfm_cfw_DFS_Sequence_v1.0.0.qr1.png](cfm_cfw_DFS_Sequence_v1.0.0.qr1.png)

### tasker

- task 관리 모듈

![cfm_v1.0.0.qr1_tasker_State.png](cfm_v1.0.0.qr1_tasker_State.png)

### remover

- 파일 삭제 요청 모듈

![cfm_v1.0.0.qr1_remover_State.png](cfm_v1.0.0.qr1_remover_State.png)

<br><br>
기존 모듈간의 Seqence Diagram
===================

### CenterFileManager, CenterFileWorker, DFS, vodSubAgent


![legacy_CenterFileManager_CenterFileWorker_DFS_vodSubAgent_Sequence.png](legacy_CenterFileManager_CenterFileWorker_DFS_vodSubAgent_Sequence.png)
