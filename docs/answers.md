# InterGold Assessment Written Answers

## Part 1: Analyze Flawed Code

### ฟังก์ชันนี้ทำอะไร

`process_gold_order` เปิด connection ไปที่ SQLite, อ่าน balance และชื่อของลูกค้าจาก `customers`, จากนั้นประมวลผลคำสั่งซื้อขายทอง ถ้าเป็น `buy` จะคำนวณ `quantity * price`, ตรวจสอบเงินพอหรือไม่ แล้วหัก balance และบันทึก order ถ้าเป็น `sell` จะเพิ่ม balance และบันทึก order ก่อน `commit` และคืนผลลัพธ์เป็น dictionary

### ปัญหาและแนวทางแก้ไข

| ปัญหา | Financial Impact | Best Practice |
|---|---|---|
| SQL Injection จากการต่อ string ใน `SELECT`, `UPDATE`, `INSERT` | ผู้โจมตีอาจแก้ balance, อ่านข้อมูลลูกค้า, หรือสร้าง order ปลอมได้ เป็นความเสียหายตรงต่อเงินและความน่าเชื่อถือของระบบ | ใช้ parameterized query เสมอ เช่น `WHERE id = ?` และส่งค่าแยกจาก SQL ห้ามใช้ f-string กับข้อมูลจากผู้ใช้ |
| ใช้ numeric แบบปกติหรือ float สำหรับราคาและยอดเงิน | Floating point ทำให้เกิด rounding error เช่นยอดเงินผิดระดับสตางค์หรือมากกว่า เมื่อรวม transaction จำนวนมากจะกระทบ reconciliation | ใช้ `Decimal` ใน Python หรือ `shopspring/decimal` ใน Go เก็บ scale ชัดเจน และกำหนด rounding policy |
| ไม่มี database transaction ที่ครอบคลุม read-check-update-insert แบบ atomic | ถ้า update balance สำเร็จแต่ insert order ล้มเหลว ระบบจะมียอดเงินเปลี่ยนแต่ไม่มี order อ้างอิง หรือกลับกัน ทำให้ ledger เพี้ยน | ใช้ transaction (`BEGIN`, `COMMIT`, `ROLLBACK`) ครอบทั้งการ validate, update balance, insert order และใช้ constraint/ledger table |
| Race condition ระหว่างอ่าน balance และ update balance | คำสั่งซื้อพร้อมกันสองรายการอาจเห็น balance เดิมทั้งคู่และหักเงินเกินยอดจริง เกิด overdraft | ใช้ row-level lock หรือ atomic conditional update เช่น `UPDATE ... SET balance = balance - ? WHERE id = ? AND balance >= ?` ภายใน transaction |
| ไม่ตรวจสอบ input ให้ครบ | `order_type`, `quantity`, `price`, `customer_id` ที่ผิดปกติ เช่น quantity ติดลบ อาจทำให้ซื้อแล้วเงินเพิ่ม หรือขายแล้วเงินลด | Validate ทุก field: order type ต้องเป็น buy/sell, quantity > 0 และเป็น step 0.5, price > 0, customer มีอยู่จริง |
| ไม่ handle กรณี `customer` เป็น `None` และไม่จัดการ exception | ระบบอาจ crash ระหว่างประมวลผล order ทำให้ผู้ใช้เห็นผลไม่ชัดเจน และ connection/transaction ค้าง | ตรวจสอบ not found, ใช้ `try/except/finally` หรือ context manager, return error ที่ชัดเจน และ log แบบ structured |
| ไม่ปิด database connection | เมื่อ traffic สูง connection/resource leak จะทำให้ระบบช้า หยุดตอบสนอง หรือ reject order ที่ถูกต้อง | ใช้ context manager หรือ connection pool ที่ lifecycle ชัดเจน |
| ใช้ `print` แทน logging/audit trail | ข้อมูลธุรกรรมอาจรั่วใน log และ audit trail ไม่ครบสำหรับตรวจสอบย้อนหลัง | ใช้ structured logging ที่ mask ข้อมูลสำคัญ และมี immutable audit/ledger event |
| Business rules กระจุกในฟังก์ชันเดียว | แก้ requirement ใหม่ เช่น spread, daily limit, price freshness ยากและเสี่ยง regression | แยก validation, pricing, repository, transaction service และเขียน unit test ต่อ rule |

## Part 4: Explain Your Decisions

### 1. Trade-offs

ผมเลือก Clean Architecture แบบบาง ไม่ทำ service framework หรือ database จริง เพราะโจทย์ต้องการ validation logic มากกว่า infrastructure จุดสำคัญคือให้ `domain` เก็บ model/result ที่เป็นแกนธุรกิจ และให้ `application` use case แยกตัวเองจาก data source ผ่าน interface เล็กๆ (`BalanceProvider`, `MarketPriceProvider`, `DailyVolumeProvider`) ทำให้โค้ดยังเรียบง่าย แต่รองรับการเปลี่ยน in-memory เป็น PostgreSQL, Redis, หรือ market price API ได้โดยไม่กระทบ domain logic

Trade-off คือมีไฟล์และ interface มากกว่าการเขียน function เดียว แต่ได้ testability และ extensibility ที่เหมาะกับระบบการเงิน ซึ่งมักมี rule เพิ่มเรื่อยๆ และต้องตรวจสอบย้อนหลังได้

### 2. Alternatives

ทางเลือกแรกคือเขียน validator เป็น function เดียวที่รับ balance และ market price เข้ามาตรงๆ วิธีนี้ง่ายที่สุด แต่เมื่อเพิ่ม spread, daily limit, customer tier, trading session, หรือ risk rule จะเริ่มบวมเร็ว

อีกทางเลือกคือทำ rule engine เต็มรูปแบบ แต่สำหรับ assessment นี้ถือว่าเกินจำเป็น เพราะ rule ยังน้อยและต้องการความอ่านง่ายมากกว่าความ dynamic ผมจึงเลือก validator service ที่มี helper function ชัดเจนและ interface สำหรับข้อมูลภายนอก

### 3. Debugging Process

ผมเริ่มจากเส้นทางเงินก่อน: balance ถูกอ่านอย่างไร, cost/revenue คำนวณอย่างไร, update balance และ insert order เป็น atomic หรือไม่ เพราะจุดนี้มี financial impact สูงสุด จากนั้นดู security boundary เช่น SQL injection และ input validation แล้วค่อยดู reliability เช่น connection lifecycle, exception handling, logging และ maintainability

การจัดลำดับ severity:

1. Critical: SQL injection, race condition, non-atomic money update
2. High: floating point/precision error, negative quantity/price, missing validation
3. Medium: crash จาก customer not found, connection leak, incomplete error handling
4. Low-to-medium: maintainability, logging style, duplicated logic

### 4. Evolution for High Throughput

ถ้าต้องรองรับหลายพันรายการต่อนาที ผมจะคง domain validator เดิมไว้ แต่เปลี่ยน infrastructure รอบนอก:

- ใช้ PostgreSQL transaction พร้อม row-level lock หรือ atomic conditional update สำหรับ balance
- แยก available balance กับ immutable ledger เพื่อ reconciliation
- ใช้ idempotency key ต่อ order เพื่อกัน retry แล้วหักเงินซ้ำ
- ใช้ message queue สำหรับ decouple order intake, validation, execution, notification และ audit
- ใช้ Redis หรือ local cache สำหรับ market price แต่ต้องมี TTL และ version/as-of timestamp
- ใช้ distributed lock เฉพาะจุดที่ต้อง serialize ต่อ customer หรือ instrument แต่ไม่ใช้แทน database transaction
- เพิ่ม observability: structured log, metrics เช่น rejection reason, latency, queue lag และ alert ตาม severity

สิ่งที่ผมจะเก็บไว้คือ decimal arithmetic, rule isolation, explicit result object, และ unit tests ต่อ business rule เพราะเป็นแกนที่ยังถูกต้องแม้ระบบโตขึ้น

### 5. Tools

ผมใช้ AI assistant เป็น pair-programming/reference tool ในบางช่วงของงาน ได้แก่ช่วยสรุปโจทย์จาก PDF, ช่วยจัดโครงเอกสารคำตอบ, และช่วยตรวจทาน checklist ว่าครอบคลุมประเด็น security, correctness, reliability และ maintainability ครบหรือไม่

สำหรับส่วน implementation ผมใช้ AI ช่วยเสนอ skeleton ของ Clean Architecture และ test cases เริ่มต้น แต่ผมเป็นคนตัดสินใจ final design เอง โดยเฉพาะ boundary ระหว่าง `domain`, `application`, และ `inmemory adapter`, การเลือกใช้ `shopspring/decimal`, rule ของ spread/daily limit, และรูปแบบ validation result

ผม verify ผลลัพธ์ด้วยการอ่านเทียบ requirement ในโจทย์อีกครั้งและรันคำสั่ง `go test ./...`, `go vet ./...`, และ `go test -race ./...` เพื่อยืนยันว่าโค้ด build ได้, test ผ่าน, และไม่มี race condition ในส่วนที่ทดสอบ สิ่งที่ผมให้ความสำคัญเป็นพิเศษคือ invariant ของระบบการเงิน: ไม่ใช้ `float64` กับจำนวนเงิน/ราคา, input ผิดต้องไม่ทำให้ระบบ crash, และ balance/daily limit ต้องคำนวณด้วย decimal อย่างสม่ำเสมอ

## Part 5: Pull Request Review Comments

### Overall

ขอบคุณที่แยก `process_batch_orders`, `process_single_order`, และ `get_batch_summary` ออกจากกัน โครงนี้อ่านง่าย และการเริ่มใช้ `Decimal` เป็นสัญญาณที่ดีสำหรับระบบการเงิน อย่างไรก็ตามก่อน merge ผมอยาก request changes ในประเด็น data integrity และ concurrency เพราะตอนนี้มีโอกาสทำให้ balance ผิดได้ภายใต้ concurrent request หรือ partial failure

### Request Changes

**Concurrency: อ่าน balance นอก lock**

ตอนนี้ `balance = customer_balances[customer_id]` ถูกอ่านก่อนเข้า `balance_lock` แต่การหักเงินเกิดใน lock ภายหลัง ทำให้สอง thread อ่าน balance เดิมพร้อมกันและทั้งคู่ผ่าน insufficient balance check ได้ ตัวอย่างเช่น balance 100,000 และมี buy สองรายการรายการละ 80,000 พร้อมกัน ทั้งคู่จะเห็นว่าเงินพอ แล้วสุดท้าย balance อาจติดลบ

ข้อเสนอ: รวม read-check-update ไว้ใน critical section เดียวกัน หรือถ้าเป็น production ให้ย้ายไปใช้ database transaction/row lock หรือ atomic conditional update เช่น update balance เฉพาะเมื่อ balance ยังพอ

**Atomicity: batch บางรายการสำเร็จบางรายการล้มเหลว**

ตอนนี้ batch ถูกประมวลผลทีละ order และ update balance ทันที ถ้ารายการที่ 1-2 สำเร็จ แต่รายการที่ 3 fail หรือ process crash กลางทาง เราจะได้ partial batch โดยไม่มี policy ชัดเจนว่าต้อง rollback หรือยอมรับ partial fill

ข้อเสนอ: ระบุ contract ของ batch ให้ชัด ถ้าต้องเป็น all-or-nothing ให้ validate ทั้ง batch ก่อน แล้วทำ update ทั้งหมดใน transaction เดียว ถ้ายอมให้ partial fill ต้องมีสถานะ batch/order ชัดเจน, audit log, และ retry semantics ที่ไม่หักเงินซ้ำ

**Precision: ยังใช้ float ใน balance, price, quantity**

แม้ import `Decimal` แล้ว แต่ `customer_balances`, `get_market_price`, และ order input ยังเป็น float (`500000.00`, `42150.00`, `quantity * price`) ซึ่งเสี่ยง rounding error ในระบบเงิน โดยเฉพาะเมื่อรวมผลหลายรายการใน batch

ข้อเสนอ: ใช้ `Decimal` ตั้งแต่ boundary ของระบบ และ parse จาก string เช่น `Decimal("42150.00")` หลีกเลี่ยงการสร้าง Decimal จาก float

**In-memory state ไม่เหมาะกับ financial source of truth**

`customer_balances` และ `order_log` เป็น global in-memory store ถ้า process restart ข้อมูลหาย และถ้ามีหลาย instance ข้อมูลจะไม่ตรงกัน นอกจากนี้ `order_log.append` ไม่ได้ lock ทำให้ concurrent write มีความเสี่ยง

ข้อเสนอ: ใช้ database เป็น source of truth พร้อม transaction และ immutable ledger table ส่วน in-memory ใช้ได้เฉพาะ test/demo เท่านั้น ถ้าต้องเก็บ cache ควรมี invalidation/TTL และไม่ใช้เป็นตัวตัดสินยอดเงินจริง

**Input validation และ error handling ยังไม่ครบ**

`order["type"]`, `order["quantity"]`, `order["price"]` และ `customer_balances[customer_id]` อาจ raise `KeyError` ได้ ถ้า client ส่ง payload ไม่ครบหรือ customer ไม่มีอยู่จริง นอกจากนี้ยังไม่ validate quantity > 0, increment 0.5, และ price > 0

ข้อเสนอ: validate schema ก่อนประมวลผล และ return rejected/error ที่ชัดเจนโดยไม่ crash

**Tolerance ของราคาไม่ตรง requirement**

โค้ดใช้ 5% แต่ requirement validation ใช้ 2% และสำหรับ buy order ต้องตรวจ expected buy price ที่รวม spread 0.5% ไม่ใช่ base market price ตรงๆ

ข้อเสนอ: centralize config เช่น `PRICE_TOLERANCE = Decimal("0.02")`, `SPREAD_RATE = Decimal("0.005")` และเขียน test ครอบ buy/sell pricing

### What I Would Approve

- การแยก function ทำให้ review และ test ง่ายขึ้น
- มีแนวคิด lock สำหรับ balance update ซึ่งเป็นทิศทางที่ถูก แต่ scope ของ lock ยังต้องครอบ read-check-update
- มี batch summary ช่วยให้ caller เห็นภาพรวมผลลัพธ์ได้ดี

### Suggested Direction

ผมแนะนำให้ refactor เป็นสอง phase: validate ทั้ง batch ด้วย Decimal และ snapshot ของ market price จากนั้น execute ภายใน transaction เดียว หากระบบต้องรองรับ partial fill ให้เพิ่ม batch status และ order status ให้ explicit พร้อม audit log เพื่อให้ finance/reconciliation ตรวจสอบได้เสมอ
