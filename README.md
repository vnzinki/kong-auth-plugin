# Auth Gateway

Trong quá trình xây dựng microservice, việc xác thực ra request (authn) cần đảm bảo được các yếu tố cơ bản:
  - Cover được toàn bộ các private API trong hệ thống
  - Scalable với số lượng lớn request
  - Dễ dàng triển khai trên nhiều service, nhiều ngôn ngữ khác nhau
  - Tránh việc duplicate các module, implement nhiều lần dẫn đến phải thay đổi hàng loạt nếu cần update
  - Tránh việc lưu key tại nhiều vị trí khác nhau, gây rủi ro bảo mật

## Edge Authentication (JWT)
Phần authn sẽ được move ra ngoài gateway. Các các edge service (service nhận request trực tiếp từ phía user) sẽ trust các header được quản lý chặt chẽ từ phía gateway.

![JWT](/docs/jwt-auth.svg)


- Target Service chỉ cần đọc header để xác thực user request, không yêu cầu implement riêng logic validate, không phụ thuộc ngôn ngữ
- Middleware cho gateway được viết bằng golang đảm bảo performance tốt cho số lượng request lớn, phục vụ việc validate key tốc độ cao, dễ dàng scale
- Publickey & Privatekey được lưu ở các vị trí khác nhau, đảm bảo tính bảo mật
- Cơ chế check blacklist tập trung, phục vụ việc blacklist token trên toàn hệ thống, có cơ chế chống lỗi (khi chết redis sẽ auto cho qua các request => các user bị ban có cơ hội sống thêm 15 phút trong trường hợp xấu nhất)

## Forward Authentication (API-KEY)

Lười viết docs quá, update sau nhé

![Forward](/docs/forward-auth.svg)

## Implement

| Header     | Description    |      Values      | Note  |
| :--------- | :------------- | :--------------: | :---: |
| x-prexfix-role | Quyền user     |   user/api/guest   |       |
| x-prexfix-uid  | Unique User ID |   StringNumber   |       |
| x-prexfix-jwt  | JWT            | Forward raw data |       |

## Example
Request của user chưa login
```
"x-prexfix-role": "guest"
"x-prexfix-uid": ""
"x-prexfix-jwt": ""
```

Request của user id 1989
```
"x-prexfix-role": "user"
"x-prexfix-uid": "1989"
"x-prexfix-jwt": "eyJhbGciOiJIU.....POk6yJV_adQssw5c"
```

## Note
  - Edge Authentication chỉ áp dụng cho edge service expose API trực tiếp cho client. Internal Service / Core Service sẽ sử dụng cơ chế khác
  - Phân quyền (AuthZ) sẽ được implement tại từng service cụ thể, ví dụ user có quyền cancel order của chỉ user đó sẽ thuộc domain của `order service`...
