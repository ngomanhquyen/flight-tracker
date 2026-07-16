# Lab 1.2 — Init + Sidecar Pattern

**Thời lượng:** ~60 phút | **CKAD domain:** Application Design and Build (20%)

## Mục tiêu

- Đọc hiểu một multi-container Pod **thật** đang chạy (app + 2 sidecar chia
  sẻ dữ liệu qua `emptyDir`) — `bot-service` trong chính project này.
- Tự viết một Pod init container + app container + sidecar từ đầu, để luyện
  phản xạ viết YAML nhanh (bot-service thật lại **không** có init container
  — lý do tại sao chính là một phần bài học).
- Tail log của một container cụ thể trong Pod nhiều container bằng
  `kubectl logs -c`.

## Chuẩn bị

Dùng lại namespace `flight-tracker`.

```bash
kubectl config set-context --current --namespace=flight-tracker
mkdir -p manifests
```

Phần A cần `bot-service` đang chạy với 2 sidecar (`cloudflared` +
`webhook-registrar`) kích hoạt — cấu hình này chỉ bật khi deploy bằng
`values-local.yaml`:

```bash
cd deployments/helm/flight-tracker
cp values-local.yaml.example values-local.yaml   # nếu chưa có
# điền postgresql.auth.postgresPassword, rabbitmq.auth.password,
# và 3 secret BOT_TELEGRAM_* trong values-local.yaml — giá trị giả cũng
# được, mục tiêu ở đây chỉ là có đủ 3 container chạy, không cần webhook
# thật sự hoạt động.
./deploy-local.sh
cd -
```

> Không muốn deploy cả stack (Postgres/RabbitMQ qua Bitnami subchart) chỉ
> để xem Phần A? Bỏ qua, làm thẳng Phần B — vẫn đủ trọn vẹn kỹ năng
> init+sidecar+`emptyDir` cần cho kỳ thi.

## Phần A — Sidecar thật: `bot-service`

`bot-service`'s Pod có 3 container khi deploy local:
`bot-service` (app), `cloudflared` (mở quick tunnel), `webhook-registrar`
(theo dõi log của `cloudflared` qua volume `tunnel-log` dùng chung, tự gọi
Telegram `setWebhook` mỗi khi có tunnel URL mới) — xem
[`templates/tunnel-registrar-configmap.yaml`](../../../deployments/helm/flight-tracker/charts/bot-service/templates/tunnel-registrar-configmap.yaml)
và [`values-local.yaml.example`](../../../deployments/helm/flight-tracker/values-local.yaml.example).

1. Tìm Pod và xác nhận nó có đúng 3 container.
2. Liệt kê toàn bộ volume của Pod này bằng `jsonpath`, tìm volume kiểu
   `emptyDir` tên `tunnel-log`.
3. Xem log riêng của từng sidecar (`-c cloudflared`, `-c webhook-registrar`)
   — không xem log gộp cả Pod.
4. Chứng minh 2 container thật sự chia sẻ dữ liệu qua volume: exec vào
   `webhook-registrar`, đọc file do `cloudflared` ghi ra.
5. Tự trả lời: vì sao container `bot-service` (container chính) **không**
   mount volume `tunnel-log`? (Gợi ý: sidecar pattern tách hẳn mối quan tâm
   phụ — ở đây là việc lộ ra ngoài Internet — khỏi logic nghiệp vụ chính;
   container chính không cần biết tunnel tồn tại.)

## Phần B — Tự viết init container

`bot-service` **cố tình không có init container** chờ `flight-service`/
`subscription-service` sẵn sàng — theo `services/bot-service/README.md`,
nó start ngay lập tức và trả lời lỗi thân thiện cho lệnh nào cần dependency
chưa lên, thay vì bị treo ở trạng thái `Init:`. Đây là một lựa chọn thiết
kế có chủ đích (graceful degradation), không phải thiếu sót — và cũng là
lý do CKAD hay hỏi "khi nào nên/không nên dùng init container". Phần này
cho bạn tự viết một Pod theo hướng ngược lại: container chính **phụ thuộc
thật sự** vào việc init container chạy xong trước.

Viết `manifests/log-pipeline.yaml` cho một Pod tên `log-pipeline` gồm:

1. Một volume `emptyDir` tên `shared-logs`, mount tại `/var/log/app` trên **cả
   3 container** bên dưới.
2. **initContainer** `init-setup`, image `busybox:1.36`, chạy một lệnh vừa
   in dòng `Init completed at <thời gian>` ra stdout **vừa** ghi dòng đó
   vào `/var/log/app/init.log` (gợi ý: `kubectl logs` chỉ đọc stdout, không
   đọc file — chỉ redirect (`>>`) vào file thì log container sẽ trống,
   dùng `tee` để làm cả hai cùng lúc), rồi kết thúc (exit 0).
3. **Container chính** `app`, image `busybox:1.36`, chạy một vòng lặp vô hạn
   ghi thêm một dòng `App tick <thời gian>` vào `/var/log/app/app.log` mỗi 5
   giây.
4. **Sidecar** `log-shipper`, image `busybox:1.36`, chạy `tail -f` trên
   `/var/log/app/app.log` để mô phỏng một log-forwarding sidecar.

Sau khi apply:

5. Theo dõi Pod chuyển trạng thái: `Init:0/1` → `Init:1/1` → `PodInitializing`
   → `Running` (ready sẽ là `2/2`, init container không tính vào số container
   ready).
6. Xem log của **init container** — dù nó đã Completed, log vẫn còn miễn Pod
   chưa bị xoá.
7. Tail log **trực tiếp** từ sidecar (`kubectl logs -c ... -f`) và xác nhận
   nó in ra đúng những dòng mà container `app` đang ghi.
8. Exec vào container `app`, đọc `/var/log/app/init.log` để xác nhận volume
   thực sự dùng chung giữa init container và container chính.

## Tiêu chí hoàn thành

**Phần A:**
- [ ] `kubectl get pod -l app.kubernetes.io/name=bot-service -o jsonpath='{.items[0].spec.containers[*].name}'` → in đủ 3 tên container
- [ ] `kubectl get pod <bot-service-pod> -o jsonpath='{.spec.volumes[*].name}'` → có `tunnel-log`
- [ ] `kubectl exec <bot-service-pod> -c webhook-registrar -- cat /shared/cloudflared.log` → đọc được nội dung (không lỗi "no such file")

**Phần B:**
- [ ] `kubectl get pod log-pipeline` → `STATUS=Running`, `READY=2/2`
- [ ] `kubectl get pod log-pipeline -o jsonpath='{.status.initContainerStatuses[0].state}'` → có `terminated` với `reason: Completed`
- [ ] `kubectl logs log-pipeline -c init-setup` → thấy dòng "Init completed at ..."
- [ ] `kubectl logs log-pipeline -c log-shipper --tail=5 -f` (Ctrl+C sau vài giây) → thấy các dòng "App tick ..." mới liên tục xuất hiện
- [ ] `kubectl exec log-pipeline -c app -- cat /var/log/app/init.log` → đọc được nội dung do init container ghi

## Dọn dẹp

```bash
kubectl delete pod log-pipeline
```
(Không xoá `bot-service` — đó là release Helm thật, không phải resource của
lab này. Nếu bạn deploy chỉ để làm Phần A và muốn gỡ luôn:
`helm uninstall flight-tracker -n flight-tracker` từ
`deployments/helm/flight-tracker`, dùng `uninstall-local.sh`.)
