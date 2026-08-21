@gobstruct("Person")
@dataclass
class Person:
    Name: str   # inferred → GOB_STRING
    Age: int    # inferred → GOB_INT

data = pygob.encode(Person(Name="Ada", Age=36))
