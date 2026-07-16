export function visible(value: string): string {
  return `visible:${value}`;
}

function hidden(value: string): string {
  if (value === "") {
    return "empty";
  }
  return value.toUpperCase();
}

export function callHidden(value: string): string {
  return hidden(value);
}
