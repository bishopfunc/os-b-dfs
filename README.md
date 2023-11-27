# os-b-dfs

## Flow
考え方
- clintとcacheはローカルにある、FSはリモートにある。
- キャッシュの更新、削除はopen時にやっておくと楽だからそうする。
- 初めてopen(file, r)するときにキャッシュする。
- FSでキャッシュを持ってるclientを記録しておく。cacheDir:{file, client}で管理する。
- open(file, w)するとき、キャッシュを持ってるclientにinvalidを送る。
- invalidを受け取ると、そのファイルを削除する。
- FSでLock状態のファイルを記録しておく。lockDir:{file, state}で管理する。
- open(file, w)するとき、requestLockを送信し、lockDirを更新して、そのファイルをロック状態にする。cacheからFSにファイルが同期されたら、requestUnLockを送信して、lockDirを更新して、ロック状態を解除する。ロック状態はreadできるがwriteできない。clientはFSにアクセスするときに、lockdirをチェックする。


### キャッシュなしread
キャッシュがない場合は直接リモートサーバを読みにいく、リモートサーバからキャッシュにコピーしておく。
```mermaid
sequenceDiagram
    participant clientA as clientA
    participant cacheA as cacheA
    participant FS as FS
    participant cacheB as cacheB
    participant clientB as clientB
    Note over FS: Files: [a.txt, b.txt]
    Note over FS: cacheDir: {<br>a.txt: [clientB], <br>b.txt: [..., ...]} 

    Note over cacheA: Files: []
    Note over cacheB: Files: [a.txt]    
    
    clientA->>cacheA: open("a.txt", r)
    cacheA->>clientA: -1
    clientA->>FS: open("a.txt", r)
    FS->>cacheA: sendFile(cacheA, "a.txt")
    Note left of FS: a.txt
    Note over cacheA: Files: [a.txt]
    FS->>FS: updateCache("a.txt", clientA)
    Note over FS: cacheDir: {<br>a.txt: [clientA, clientB], <br>b.txt: [..., ...]} 
    FS->>clientA: fd
    clientA->>clientA: bytes = read(fd, buf)
    clientA->>FS: close(fd)
    FS->>FS: closeFile(fd)
```
### キャッシュありread
キャッシュあるなら、キャッシュを読みにいく、一番簡単
```mermaid
sequenceDiagram
    participant clientA as clientA
    participant cacheA as cacheA
    participant FS as FS
    participant cacheB as cacheB
    participant clientB as clientB
    Note over FS: Files: [a.txt, b.txt]
    Note over FS: cacheDir: {<br>a.txt: [clientA, clientB], <br>b.txt: [..., ...]} 

    Note over cacheA: Files: [a.txt]    
    Note over cacheB: Files: [a.txt]    

    clientA->>cacheA: open("a.txt", r)
    cacheA->>cacheA: fd = openFile("a.txt")
    cacheA->>clientA: fd
    Note left of cacheA: a.txt
    FS->>clientA: fd
    clientA->>clientA: bytes = read(fd, buf)
    clientA->>FS: close(fd)
    FS->>FS: closeFile(fd)
```

### キャッシュなしwrite
キャッシュがないなら、直接リモートサーバを書き換える。open時にキャッシュの無効化とファイルロックを行う。close後にファイルロックを解除する。
```mermaid
sequenceDiagram
    participant clientA as clientA
    participant cacheA as cacheA
    participant FS as FS
    participant cacheB as cacheB
    participant clientB as clientB
    Note over FS: Files: [a.txt, b.txt]
    Note over FS: cacheDir: {<br>a.txt: [clientB], <br>b.txt: [..., ...]} 
    Note over cacheA: Files: [a.txt]    
    Note over cacheB: Files: [a.txt] 

    clientA->>cacheA: open("a.txt", w)
    cacheA->>clientA: -1
    clientA->>FS: open("a.txt", w)
    clientA->>FS: requestLock("a.txt")
    FS->>FS :updateLock()
    Note over FS: lockDir: {<br>a.txt: True, <br>b.txt: False} 

    FS->>FS: fd = openFile("a.txt")
    FS->>cacheB: sendInvalid(FS, cacheB, "a.txt")
    cacheB->>cacheB: deleteFile("a.txt")
    Note over cacheB: Files: []

    FS->>FS: deleteCache("a.txt")
    Note over FS: cacheDir: {<br>a.txt: [], <br>b.txt: [..., ...]} 
    FS->>clientA: fd
    clientA->>clientA: bytes = write(fd, buf)

    Note over cacheB: Lock時readの動作
    clientB->>FS: open("a.txt", w)
    FS->>FS: True = checkLock()
    FS->>clientB: -1
    clientB->>FS: open("a.txt", r)
    FS->>FS: True = checkLock()
    FS->>FS: fd = openFile("a.txt", r)
    FS->>clientB: fd
    Note over cacheB: 終了

    clientA->>FS: close(fd)
    FS->>FS: closeFile(fd)

    clientA->>FS: requestUnLock("a.txt")
    FS->>FS :updateLock()
    Note over FS: lockDir: {<br>a.txt: False, <br>b.txt: False} 

```

### キャッシュありwrite
キャッシュがあるなら、キャッシュの内容を書き換える。open時にキャッシュの無効化とファイルロックを行う。closeしたのはキャッシュにあるファイルなので、キャッシュからリモートサーバにコピーしてから、ファイルロックを解除する。

```mermaid
sequenceDiagram
    participant clientA as clientA
    participant cacheA as cacheA
    participant FS as FS
    participant cacheB as cacheB
    participant clientB as clientB
    Note over FS: Files: [a.txt, b.txt]
    Note over FS: cacheDir: {<br>a.txt: [clientB], <br>b.txt: [..., ...]} 
    Note over cacheA: Files: [a.txt]    
    Note over cacheB: Files: [a.txt, b.txt] 

    clientA->>cacheA: open("a.txt", w)
    clientA->>FS: requestLock("a.txt")
    FS->>FS :updateLock()
    Note over FS: lockDir: {<br>a.txt: True, <br>b.txt: False} 

    FS->>FS: deleteFile(cacheA, "a.txt")
    FS-xcacheA: sendInvalid(FS, cacheA, "a.txt")
    FS->>cacheB: sendInvalid(FS, cacheB, "a.txt")
    cacheB->>cacheB: deleteFile("a.txt")
    Note over cacheB: Files: [b.txt]
    FS->>FS: deleteCache("a.txt")
    Note over FS: cacheDir: {<br>a.txt: [], <br>b.txt: [..., ...]} 

    cacheA->>clientA: fd
    clientA->>clientA: bytes = write(fd, newData)
    clientA->>cacheA: close(fd)
    cacheA->>cacheA: closeFile("a.txt")
    Note over cacheA: Files: [a.txt]

    cacheA->>FS: sendFile(cacheA, FS, "a.txt")
    Note over FS: Files: [a.txt]

    Note over cacheB: Lock時readの動作(省略)
    clientB->>FS: 

    clientA->>FS: requestUnLock("a.txt")
    FS->>FS :updateLock()
    Note over FS: lockDir: {<br>a.txt: False, <br>b.txt: False} 

```
