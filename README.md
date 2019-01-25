# [Golang Projects]  

## [Folder Description] 

### [ConnProxy 正/反代理工具]  
This is used go language to develope program of proxy server with support http and https.   
使用GO 语言开发的一个程序，它可以创建代理访问web服务器的工具，多级衔接或穿透配置，正向/反向，支持原始的TCP以及HTTP和HTTPS。  

===Propertys and functions===   
* 1.can custom config to manager the proxy server(timeout,conn num,wait num,filter ip,etc...).  
* 2.can set direct proxy link request.  
* 3.can set reverse proxy server list and auto switch reverse proxy server or custom function to handle switch plot.  
* 4.support queue wait,queue timeout,connection num control,ip filter.
* 5.support multiple connection accept for high request.  
* 6.support timeout model,auto analysis close time or absolute deadline.  
* 7.support long time file download.  
* 8.etc....

* 1.可以自定义配置进行代理工具的设置，目前包含 超时，最大并发连接数，最大等待数，IP过滤，等等  
* 2.支持直接代理访问  
* 3.支持多反向代理服务器设置，类似于nginx代理切换功能。  
* 4.自动切换反向配置的代理服务器，也可以自己实现切换策略  
* 5.支持大并发连接。  
* 6.支持超时模式，智能确定关闭时间和强制过期时间。  
* 7.支持长时间下载大文件。  
* 8.等等....

### [GOBLog go 日志系统]  log record for golang program
This is log writer for go program runing.

===Propertys and functions===
* 1.can custom formatter message, default support simple and pattern Formatter.
* 2.can custom appender output and multiple Output Appenders.
* 3.can set logger output level , support level filter.
* 4.can support logger factory pool manager.
* 5.can support multiple thread log writer.
* 6.can support rotate file config. 
