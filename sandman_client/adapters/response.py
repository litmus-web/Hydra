from typing import Union, Optional

from ..codes import OpCodes


class OutGoingResponse:
    def __init__(
            self,
            req_id: int,
            status: Optional[int] = None,
            headers: Optional[Union[list, tuple]] = None,
            body: Optional[str] = None
    ):
        if req_id is not None:
            assert type(req_id) == int, "req_id type is not int"  # The server only really cares about this type.
        if status is not None:
            assert type(status) == int, "status type is not int"  # The server only really cares about this type.

        self.req_id = req_id
        self.status = status
        self.headers = headers
        if body is None:
            self.body = ""
        else:
            self.body = body

    def to_dict(self):
        return {
            "op": OpCodes.HTTP_REQUEST,
            "request_id": self.req_id,
            "status": self.status,
            "headers": self.headers,
            "body": self.body
        }

    def __repr__(self):
        return "OutGoingResponse(request_id={}, status_code={})".format(self.req_id, self.status)
