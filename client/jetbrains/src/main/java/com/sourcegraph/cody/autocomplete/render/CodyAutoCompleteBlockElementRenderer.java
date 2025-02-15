package com.sourcegraph.cody.autocomplete.render;

import com.intellij.openapi.editor.Editor;
import com.intellij.openapi.editor.Inlay;
import com.intellij.openapi.editor.impl.EditorImpl;
import com.intellij.openapi.editor.markup.TextAttributes;
import java.awt.*;
import java.util.Comparator;
import java.util.stream.Collectors;
import org.jetbrains.annotations.NotNull;

/** Implements the logic to render an autocomplete item inline in the editor. */
public class CodyAutoCompleteBlockElementRenderer extends CodyAutoCompleteElementRenderer {

  public CodyAutoCompleteBlockElementRenderer(@NotNull String text, @NotNull Editor editor) {
    super(text, editor, AutoCompleteRendererType.BLOCK);
  }

  @Override
  public int calcWidthInPixels(@NotNull Inlay inlay) {
    EditorImpl editor = (EditorImpl) inlay.getEditor();
    String longestLine = text.lines().max(Comparator.comparingInt(String::length)).orElse("");
    return editor.getFontMetrics(Font.PLAIN).stringWidth(longestLine);
  }

  @Override
  public int calcHeightInPixels(@NotNull Inlay inlay) {
    int lineHeight = inlay.getEditor().getLineHeight();
    int linesCount = (int) text.lines().count();
    return lineHeight * linesCount;
  }

  @Override
  public void paint(
      @NotNull Inlay inlay,
      @NotNull Graphics g,
      @NotNull Rectangle targetRegion,
      @NotNull TextAttributes textAttributes) {
    g.setFont(getFont());
    g.setColor(this.themeAttributes.getForegroundColor());
    int x = targetRegion.x;
    int baseYOffset = fontYOffset();
    int i = 0;
    for (String line : this.text.lines().collect(Collectors.toList())) {
      int y = targetRegion.y + baseYOffset + (i * this.editor.getLineHeight());
      g.drawString(line, x, y);
      i++;
    }
  }

  @Override
  public String toString() {
    return "CodyAutoCompleteBlockElementRenderer{" + "text='" + text + '\'' + '}';
  }
}
