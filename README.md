# airtask
airtask is task system, support cmdline, cmd file and plugin(module,like:hello.so).


airtask develop on airfk. 

* [airfk](https://github.com/yuleihua/airfk) : airman framework for micro-service. Based on ethereum, a number of new features have been added.


### 1. protocol:
JSON RPC API
JSON is a lightweight data-interchange format. It can represent numbers, strings, ordered sequences of values, and collections of name/value pairs.

JSON-RPC is a stateless, light-weight remote procedure call (RPC) protocol. Primarily this specification defines several data structures and the rules around their processing. It is transport agnostic in that the concepts can be used within the same process, over sockets, over HTTP, or in many various message passing environments. It uses JSON (RFC 4627) as data format.

### 2. API

#### 2.1 args
most of arguments in JobArgs.
```
// Job is task job.
type JobArgs struct {
	Name     *string        `json:"name"`
	Extra    *hexutil.Bytes `json:"extra"`
	Type     *string        `json:"type"`
	UUID     uint64         `json:"uuid"`
	Datetime int64          `json:"datetime"`
	Retry    int            `json:"retry"`
	Interval int            `json:"interval"`
}

```

#### 2.2 add task api
##### 2.2.1 cmdline mode

**request**

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_addTask","params":[{"name":"dev", "type":"cmd", "interval":5, "extra":"0x756e616d65202d61"}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":362666528966967296}
 ```
 
 if you want get result, api is like:
 
 ```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_getResult","params":[{"name":"dev","uuid":362666528966967296}],"id":67}' http://127.0.0.1:5050
{"jsonrpc":"2.0","id":67,"result":{"info":"{\"id\":362666528966967296,\"begin_time\":1561269331,\"end_time\":1561269331,\"error\":\"success\",\"output\":\"0x44617277696e2068656c6c6f6b612e6c6f63616c2031382e362e302044617277696e204b65726e656c2056657273696f6e2031382e362e303a20546875204170722032352032333a31363a32372050445420323031393b20726f6f743a786e752d343930332e3236312e347e322f52454c454153455f5838365f3634207838365f36340a\"}"}}
 ```

##### 2.2.2 cmd file mode

**request**

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_addTask","params":[{"name":"dev", "type":"sh", "interval":5, "extra":"0x756e616d65202d61203e3e202f746d702f756e61656875612e747874"}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":362669502707531776}
 ```

##### 2.2.3 plugin mode

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_addTask","params":[{"name":"dev", "type":"plugin", "interval":5, "extra":"0x68656c6c6f40302e302e31"}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":362669774569734144}
 ```
 
#### 2.3 get task api

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_getTask","params":[{"name":"dev","uuid":362450735830401024}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":{"circle":0,"index":98,"info":"{\"name\":\"dev\",\"type\":\"cmd\",\"uuid\":\"0x0507af061dc00000\",\"retry\":1,\"interval\":50,\"add_time\":1561217877,\"limit_time\":0,\"extra\":\"0x6c73202d6c202f746d70\"}"}}
 ```

#### 2.4 check task api

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_checkTask","params":[{"name":"dev","uuid":362450735830401024}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":true}
 ```
 
#### 2.5 get result api

```
 curl -H "Content-Type: application/json"  -X POST --data '{"jsonrpc":"2.0","method":"task_getResult","params":[{"name":"dev","uuid":362666528966967296}],"id":67}' http://127.0.0.1:5050
```
**reponse**
 
 ```
 {"jsonrpc":"2.0","id":67,"result":{"info":"{\"id\":362666528966967296,\"begin_time\":1561269331,\"end_time\":1561269331,\"error\":\"success\",\"output\":\"0x44617277696e2068656c6c6f6b612e6c6f63616c2031382e362e302044617277696e204b65726e656c2056657273696f6e2031382e362e303a20546875204170722032352032333a31363a32372050445420323031393b20726f6f743a786e752d343930332e3236312e347e322f52454c454153455f5838365f3634207838365f36340a\"}"}}
 ```
 
### 3. subscribe

#### 3.1 protocol


#### 3.2 new task:
##### 3.2.1 subscribe

```
{"jsonrpc": "2.0", "id": 1, "method": "task_subscribe", "params": ["newTask"]}
{"jsonrpc":"2.0","id":1,"result":"0x8dbf125a72771145b4109b5daa187dce"}
```

##### 3.2.2 publish
 ```
{"jsonrpc":"2.0","method":"task_subscription","params":{"subscription":"0x8dbf125a72771145b4109b5daa187dce","result":362662705888231424}}
 ```
 
#### 3.2.3 unsubscribe

```
{"id": 1, "method": "task_unsubscribe", "params": ["0x88ed423375b0550e5819095bc56c31d0"]}
{"jsonrpc":"2.0","id":1,"result":true}
```
 
#### 3.3 task results:
##### 3.3.1 subscribe

```
{"jsonrpc": "2.0", "id": 1, "method": "task_subscribe", "params": ["results"]}
{"jsonrpc":"2.0","id":1,"result":"0x88ed423375b0550e5819095bc56c31d0"}
```

##### 3.3.2 publish
 ```
{"jsonrpc":"2.0","method":"task_subscription","params":{"subscription":"0x88ed423375b0550e5819095bc56c31d0","result":{"id":362673803127422976,"begin_time":1561271066,"end_time":1561271066,"error":"success","output":"0x44617277696e2068656c6c6f6b612e6c6f63616c2031382e362e302044617277696e204b65726e656c2056657273696f6e2031382e362e303a20546875204170722032352032333a31363a32372050445420323031393b20726f6f743a786e752d343930332e3236312e347e322f52454c454153455f5838365f3634207838365f36340a"}}}
 ```
#### 3.3.3 unsubscribe

```
{"id": 1, "method": "task_unsubscribe", "params": ["0x88ed423375b0550e5819095bc56c31d0"]}
{"jsonrpc":"2.0","id":1,"result":true}
```

### 4. service registration and discovery
* etcd    
* consul 

Future support for zookeeper.

## Q&A
   * email: huayulei_2003@hotmail.com
   * QQ: 290692402
   
   