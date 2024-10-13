
API

```
头部数据包含 流水号、命令字、包类型、压缩包类型、包长度、数据长度、数据内容
响应数据包含 流水号、命令字、包类型、压缩包类型、包长度、数据长度、数据内容
```

解析
```
通过协议头的解析，获取长度、获取数据，数据解压成标准的byte数据，二次封装为标准对象。
数据的格式是 小端在前的GBK格式。
根据 命令字 以及流水号 实现多线程异步处理，命令字可知道是什么请求，流水号可以进行业务处理。
压缩包的解压方式为 Inflater 类解压响应内容会携带通达信标准协议字段，用来区分协议的类型。

```

连接
```
socket连接上后需要进行2次连接
发送内容为监听招商证券的连接的二进制数据
连接成功后需要发送心跳连接（用来判断连接是否正常）
```


通信
```
正式建立连接后可以通信，可以建立多个socket同时通信
socket的端口和地址 在通达信的主站行情中可以获取命令字
```


```
public int LOGIN_ONE                            = 0x000d;//第一次登录
public int LOGIN_TWO                            = 0x0fdb;//第二次登录
public int HEART                                = 0x0004;//心跳维持
public int STOCK_COUNT                          = 0x044e;//股票数目
public int STOCK_LIST                           = 0x0450;//股票列表
public int KMINUTE                              = 0x0537;//当天分时K线
public int KMINUTE_OLD                          = 0x0fb4;//指定日期分时K线
public int KLINE                                = 0x052d;//股票K线
public int BIDD                                 = 0x056a;//当日的竞价
public int QUOTE                                = 0x053e;//实时五笔报价
public int QUOTE_SORT                           = 0x053e;//沪深排序
public int TRANSACTION                          = 0x0fc5;//分笔成交明细
public int TRANSACTION_OLD                      = 0x0fb5;//历史分笔成交明细
public int FINANCE                              = 0x0010;//财务数据
public int COMPANY                              = 0x02d0;//公司数据  F10
public int EXDIVIDEND                           = 0x000f;//除权除息
public int FILE_DIRECTORY                       = 0x02cf;//公司文件目录
public int FILE_CONTENT                         = 0x02d0;//公司文件内容
```
