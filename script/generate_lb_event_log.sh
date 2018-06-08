#!/bin/sh

if [ $# -ne 1 ];then
	echo "Usage:$0 <file name>"
	exit 1
fi

FILE=$1

T=$(date +%s)
T2=$(date -d@${T} +%Y%m%d)
LOG=EventLog[${T2}].log
echo "0x40ffff,1,${T},Server 125.159.40.3 Selected for Client StreamID : 80e111ea-e65c-4099-9cc5-d02ad8019b8a, ClientID : 0, GLB IP : 125.159.40.5's file(${FILE}) Request" >> $LOG
echo "0x40ffff,1,${T},Server 125.159.40.3 Selected for Client StreamID : 3f004a6e-82af-4dce-85ba-9bbf9c7cb8cb, ClientID : 0, GLB IP : 125.144.96.6's file(${FILE}) Request" >> $LOG
echo "0x40ffff,4,${T},File Not Found, UUID : fffb233a-376a-4c2f-842e-553fb68af9cf, GLB IP : 125.144.161.6, MV6F9001SGL1500001_K20150909214818.mpg" >> $LOG
echo "0x40ffff,1,${T},Server 125.144.91.87 Selected for Client StreamID : 360527d4-44b3-4b8f-aef7-dbf8fd230d54, ClientID : 0, GLB IP : 125.144.169.6's file(M33E80DTSGL1500001_K20141022144006.mpg) Request" >> $LOG
echo "0x40ffff,1,${T},Server 125.159.40.3 Selected for Client StreamID : c93a7db2-ccaf-4765-af8d-7ddc2d33a812, ClientID : 0, GLB IP : 125.159.40.5's file(${FILE}) Request" >> $LOG
echo "0x40ffff,4,${T},File Not Found, UUID : f1add5cf-75ac-41ab-a6ff-85d9e0927762, GLB IP : 125.144.169.6, MK4E7BK2SGL0800014_K20120725124707.mpg" >> $LOG
echo "0x40ffff,1,${T},Server 125.144.97.67 Selected for Client StreamID : 06fb572e-7602-4231-8670-cb6526603fb0, ClientID : 0, GLB IP : 125.146.8.6's file(M33H90E2SGL1500001_K20171008222635.mpg) Request" >> $LOG
echo "0x40ffff,1,${T},Server 125.159.40.3 Selected for Client StreamID : 3c61af91-cd6a-4dd6-bc04-5ec6bc78b94f, ClientID : 0, GLB IP : 125.159.40.5's file(${FILE}) Request" >> $LOG
echo "0x40ffff,1,${T},Server 125.159.40.3 Selected for Client StreamID : 97096b41-afe1-44d8-b57c-e758a70883d9, GLB IP : 125.159.40.5's file(${FILE}) Request" >> $LOG
