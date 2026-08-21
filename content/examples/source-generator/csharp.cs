[GobStruct("Point")]        // the Go type name that goes on the wire
public partial class Point  // `partial` lets the source generator emit the schema
{                           // at compile time — no reflection, AOT-friendly
    public long X { get; set; }
    public long Y { get; set; }
}
