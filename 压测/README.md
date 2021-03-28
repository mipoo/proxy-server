#### 配件
1. 本机redis实例，用于同redis基准压测比对
2. gnet_serv , 实现yyp跟redis#get，充当后端服务
3. proxy-server。配置项：是否使用0拷贝；本机端口，对端地址
4. net_proxy。配置项：对端地址