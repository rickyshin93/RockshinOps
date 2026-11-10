function textParts(text: string) {
  return text.split(/([,万亿兆])/).filter(Boolean).map(part => ({
    text: part,
    grouping: /^[,万亿兆]$/.test(part),
    className: /^[万亿兆]$/.test(part) ? "calculation-group-unit" : undefined,
  }));
}

// History, results and references share the same visible number typography.
export function CalculationText({ text }: { text: string }) {
  return textParts(text).map((part, index) => part.className
    ? <span className={part.className} key={index}>{part.text}</span> : part.text);
}

// Editable numeric tokens decorate their raw digits with grouping marks;
// reference chips retain visible text for selection and their accessible label.
export function paintCalculationText(node: HTMLElement, text: string, decorativeGrouping = false) {
  const children = textParts(text).map(part => {
    if (!part.className && !(decorativeGrouping && part.grouping)) return document.createTextNode(part.text);
    const span = document.createElement("span");
    if (part.className) span.className = part.className;
    if (decorativeGrouping && part.grouping) {
      span.dataset.calculationGrouping = part.text;
      span.contentEditable = "false";
      span.setAttribute("aria-hidden", "true");
    } else span.textContent = part.text;
    return span;
  });
  node.replaceChildren(...children);
}
