# gracefullDemo
热重启fork部分演示

1开一个终端执行 ./main
2再开另外一个终端 执行 kill -HUP [pid] ，pid在第一步可以看到，或者ps -ef | grep "main" 也可以看到
可以看到现在服务的是子进程了，同时也可以开siege去压测，服务不会断掉。

