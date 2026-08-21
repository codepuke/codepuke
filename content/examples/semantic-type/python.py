# Go: type Status string
class Status(enum.Enum):
    ACTIVE = "active"
    INACTIVE = "inactive"

GOB_STATUS = SemanticType(
    wire_type=GOB_STRING,
    encode=lambda status: status.value,   # Status → str for the wire
    decode=Status,                        # str → Status
    zero=Status.INACTIVE,
)

UserSchema = Schema("User", Name=GOB_STRING, Status=GOB_STATUS)

data = pygob.encode({"Name": "Ada", "Status": Status.ACTIVE}, schema=UserSchema)

# Nothing on the wire marks the field as a Status, so without a schema it
# decodes back as the underlying primitive.
user = pygob.decode(data)
print(user.Status)   # "active"
