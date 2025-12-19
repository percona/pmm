const kebabize = (str: string) =>
  str
    .replace(
      /[A-Z]+(?![a-z])|[A-Z]/g,
      ($, ofs) => (ofs ? "-" : "") + $.toLowerCase()
    )
    .replace(/\s+(?=[-])/, "")
    .replace(/\s+(?![-])/, "-");

export { kebabize };
