import cv2
import numpy as np
import sys
from pathlib import Path

def preprocess_image(input_path: str, output_dir: str = "processed"):
    """Предобработка одной страницы (без разделения разворота)."""
    img = cv2.imread(input_path)
    if img is None:
        raise ValueError(f"Не удалось прочитать изображение: {input_path}")
    
    # Увеличиваем разрешение
    height, width = img.shape[:2]
    if width < 1500:
        scale = 1500 / width
        img = cv2.resize(img, None, fx=scale, fy=scale, interpolation=cv2.INTER_CUBIC)
    
    # ВАЖНО: Если изображение вертикальное - оставляем как есть
    # Если горизонтальное и очень широкое (aspect_ratio > 2.0) - вероятно это разворот
    # Но для безопасности всё равно НЕ делим, пусть Yandex сам разбирается
    aspect_ratio = img.shape[1] / img.shape[0]
    if aspect_ratio > 2.0:
        print(f"Обнаружен широкий кадр (ratio={aspect_ratio:.2f}). "
              f"Пожалуйста, фотографируйте по одной странице!")
    
    # Добавляем белые отступы
    border_size = 20
    img = cv2.copyMakeBorder(
        img, border_size, border_size, border_size, border_size,
        cv2.BORDER_CONSTANT, value=[255, 255, 255]
    )
    
    # Grayscale
    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    
    # Удаляем тени
    kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (20, 20))
    bg = cv2.morphologyEx(gray, cv2.MORPH_DILATE, kernel)
    diff = cv2.absdiff(bg, gray)
    shadow_removed = 255 - diff
    
    # Ослабляем линии тетради
    horizontal_kernel = cv2.getStructuringElement(cv2.MORPH_RECT, (30, 1))
    lines = cv2.morphologyEx(shadow_removed, cv2.MORPH_OPEN, horizontal_kernel, iterations=1)
    no_lines = cv2.addWeighted(shadow_removed, 0.8, shadow_removed - lines, 0.2, 0)
    no_lines = np.clip(no_lines, 0, 255).astype(np.uint8)
    
    # Улучшаем контраст
    clahe = cv2.createCLAHE(clipLimit=2.0, tileGridSize=(8, 8))
    enhanced = clahe.apply(no_lines)
    
    # Создаем директорию
    output_path = Path(output_dir)
    output_path.mkdir(exist_ok=True)
    
    # Всегда сохраняем как ОДНУ страницу
    base_name = Path(input_path).stem
    single_path = output_path / f"{base_name}_processed.jpg"
    cv2.imwrite(str(single_path), enhanced, [cv2.IMWRITE_JPEG_QUALITY, 95])
    
    print(f"Обработана одна страница")
    return [str(single_path)]

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Использование: python preprocess.py <путь_к_изображению>")
        sys.exit(1)
    
    input_file = sys.argv[1]
    output_files = preprocess_image(input_file)
    
    for f in output_files:
        print(f"OUTPUT:{f}")