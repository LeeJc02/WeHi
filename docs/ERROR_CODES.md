# Error Codes

## Auth

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `INVALID_ARGUMENT` | `400` | Registration arguments do not meet minimum requirements. |
| `USERNAME_ALREADY_EXISTS` | `409` | Username is already registered. |
| `INVALID_CREDENTIALS` | `401` | Username or password is invalid. |
| `INVALID_REFRESH_TOKEN` | `401` | Refresh token is invalid or expired. |
| `INVALID_ACCESS_TOKEN` | `401` | Access token is invalid or expired. |

## Friend

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `INVALID_FRIEND_REQUEST_TARGET` | `400` | The friend request target is invalid. |
| `TARGET_USER_NOT_FOUND` | `404` | The target user does not exist. |
| `FRIENDSHIP_ALREADY_EXISTS` | `409` | The friendship already exists. |
| `FRIEND_REQUEST_ALREADY_PENDING` | `409` | There is already a pending friend request. |
| `FORBIDDEN_FRIEND_REQUEST_ACTION` | `403` | Current user cannot approve/reject this friend request. |
| `FRIEND_REQUEST_NOT_PENDING` | `409` | The friend request is no longer pending. |

## Conversation

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `INVALID_DIRECT_CONVERSATION_TARGET` | `400` | Direct conversation target is invalid. |
| `GROUP_NAME_REQUIRED` | `400` | Group conversation requires a name. |
| `GROUP_MEMBER_COUNT_INVALID` | `400` | Group conversation requires enough distinct members. |
| `GROUP_MEMBER_NOT_FOUND` | `404` | One of the requested group members does not exist. |
| `CONVERSATION_MEMBERSHIP_REQUIRED` | `403` | Current user is not a member of the conversation. |
| `FORBIDDEN_CONVERSATION_ACTION` | `403` | Current user lacks permission for this conversation action. |
| `CONVERSATION_MEMBER_NOT_FOUND` | `404` | The target conversation member does not exist. |

## Message

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `INVALID_CURSOR` | `400` | Cursor format is invalid. |
| `UNSUPPORTED_MESSAGE_TYPE` | `400` | Message type is not supported. |
| `MESSAGE_CONTENT_REQUIRED` | `400` | Message content is required. |

## Search

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `SEARCH_UNAVAILABLE` | `502` | Search backend is not configured or unavailable. |

## Generic

| Error Code | HTTP Status | Meaning |
| --- | --- | --- |
| `INTERNAL_ERROR` | `500` | Unclassified internal error. |
