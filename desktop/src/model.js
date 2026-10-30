export function reconcileBuffer(buffer, disk) {
  if (!buffer || buffer.content === buffer.base) return { ...disk, base: disk.content, conflict: null };
  if (buffer.revision === disk.revision) return buffer;
  return { ...buffer, conflict: disk };
}
export function selectedAfterRefresh(slides, selected, previousIndex = 0) {
  return slides.find(s => s.id === selected)?.id || slides[Math.min(previousIndex, slides.length - 1)]?.id || '';
}
export function isDirty(buffer) { return !!buffer && buffer.content !== buffer.base; }

// Fit a complete slide inside the frame, leaving its border unobscured.
export function fitSlideRect(rect, inset = 1) {
  const width=Math.max(0,rect.width-inset*2), height=Math.max(0,rect.height-inset*2);
  const scale=Math.min(width/1280,height/720);
  const fittedWidth=1280*scale, fittedHeight=720*scale;
  return {x:rect.x+(rect.width-fittedWidth)/2,y:rect.y+(rect.height-fittedHeight)/2,width:fittedWidth,height:fittedHeight};
}
