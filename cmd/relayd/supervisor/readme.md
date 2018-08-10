# supervisor

## 使用说明书

http://supervisord.org/installing.html

## 安装依赖

```
sudo pip install --upgrade pip
sudo pip install -r requirements.txt
```

## supervisord 管理

```
http://supervisord.org/running.html#supervisorctl-actions

supervisord 安装完成后有两个可用的命令行 supervisord 和 supervisorctl，命令使用解释如下：

supervisord，初始启动 Supervisord，启动、管理配置中设置的进程。
supervisorctl stop programxxx，停止某一个进程(programxxx)，programxxx 为 [program:beepkg] 里配置的值，这个示例就是 beepkg。
supervisorctl start programxxx，启动某个进程
supervisorctl restart programxxx，重启某个进程
supervisorctl stop groupworker: ，重启所有属于名为 groupworker 这个分组的进程(start,restart 同理)
supervisorctl stop all，停止全部进程，注：start、restart、stop 都不会载入最新的配置文件。
supervisorctl reload，载入最新的配置文件，停止原有进程并按新的配置启动、管理所有进程。
supervisorctl update，根据最新的配置文件，启动新配置或有改动的进程，配置没有改动的进程不会受影响而重启。

注意：显示用 stop 停止掉的进程，用 reload 或者 update 都不会自动重启。
```

## 安装relayd

```
echo_supervisord_conf > supervisord.conf
sudo mv echo_supervisord_conf > /etc/supervisord.conf
sudo mkdir /etc/supervisord.conf.d
sudo echo "[include]" >> /etc/supervisord.conf
sudo echo "files = /etc/supervisord.conf.d/*.conf" >> /etc/supervisord.conf
sudo cp supervisord_relayd.conf /etc/supervisord.conf.d/
```

