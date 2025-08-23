#!/bin/bash

# Название выходного файла
OUTPUT_FILE="project_snapshot.txt"

# Файлы для исключения из сбора (сам скрипт и результат)
EXCLUDE_FILES=(
    "project_collector.sh"
    "project_snapshot.txt"
)

# Временный файл для хранения списка файлов
TEMP_FILE_LIST="/tmp/project_files_$.txt"

# Цвета для вывода в консоль
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Директории и файлы для исключения
EXCLUDE_DIRS=(
    ".git"
    "node_modules"
    ".next"
    "dist"
    "build"
    "coverage"
    ".pytest_cache"
    "__pycache__"
    ".venv"
    "venv"
    "env"
    ".env"
    "vendor"
    ".idea"
    ".vscode"
    ".DS_Store"
    "*.pyc"
    "*.pyo"
    "*.egg-info"
    ".gradle"
    "target"
)

# Бинарные расширения файлов для исключения
BINARY_EXTENSIONS=(
    "jpg" "jpeg" "png" "gif" "bmp" "ico" "svg" "webp"
    "pdf" "doc" "docx" "xls" "xlsx" "ppt" "pptx"
    "zip" "tar" "gz" "rar" "7z"
    "exe" "dll" "so" "dylib"
    "mp3" "mp4" "avi" "mov" "wmv"
    "ttf" "otf" "woff" "woff2"
    "sqlite" "db"
    "jar" "war" "ear"
    "bin" "dat"
    "lock"
)

# Функция для проверки, является ли файл бинарным по расширению
is_binary_by_extension() {
    local file="$1"
    local extension="${file##*.}"
    # Преобразование в нижний регистр для совместимости с macOS
    extension=$(echo "$extension" | tr '[:upper:]' '[:lower:]')
    
    for ext in "${BINARY_EXTENSIONS[@]}"; do
        if [[ "$extension" == "$ext" ]]; then
            return 0
        fi
    done
    return 1
}

# Функция для проверки, является ли файл текстовым
is_text_file() {
    local file="$1"
    
    # Сначала проверяем по расширению
    if is_binary_by_extension "$file"; then
        return 1
    fi
    
    # Проверяем MIME-тип файла
    if command -v file >/dev/null 2>&1; then
        file_type=$(file -b --mime-type "$file" 2>/dev/null)
        case "$file_type" in
            text/*|application/json|application/xml|application/javascript|application/x-httpd-php)
                return 0
                ;;
            *)
                # Дополнительная проверка для файлов без расширения или с неопределенным типом
                if file -b "$file" 2>/dev/null | grep -q "text\|ASCII\|UTF"; then
                    return 0
                fi
                return 1
                ;;
        esac
    fi
    
    return 0
}

# Функция для создания строки исключений для find
create_find_excludes() {
    local excludes=""
    for dir in "${EXCLUDE_DIRS[@]}"; do
        excludes="$excludes -path './$dir' -prune -o"
    done
    echo "$excludes"
}

# Функция для подсчета файлов
count_files() {
    local count=0
    while IFS= read -r file; do
        # Пропускаем файлы из списка исключений
        local skip=false
        for exclude in "${EXCLUDE_FILES[@]}"; do
            if [[ "$file" == "./$exclude" ]]; then
                skip=true
                break
            fi
        done
        if [[ "$skip" == true ]]; then
            continue
        fi
        
        if [[ -f "$file" ]] && is_text_file "$file"; then
            ((count++))
        fi
    done < "$TEMP_FILE_LIST"
    echo "$count"
}

# Функция для генерации дерева директорий
generate_tree() {
    echo "СТРУКТУРА ПРОЕКТА"
    echo "================================================================================"
    echo ""
    
    # Проверяем наличие команды tree
    if command -v tree >/dev/null 2>&1; then
        # Создаем строку исключений для tree
        local tree_ignores=""
        for dir in "${EXCLUDE_DIRS[@]}"; do
            tree_ignores="$tree_ignores -I '$dir'"
        done
        
        # Используем tree с исключениями
        eval "tree -a $tree_ignores --dirsfirst"
    else
        # Если tree не установлен, используем альтернативный метод
        echo "Установите 'tree' для лучшего отображения: brew install tree"
        echo ""
        find . -type d -not -path '*/\.*' | sed 's|^\./||' | sort
    fi
    
    echo ""
    echo "================================================================================"
    echo ""
}

# Функция для обработки одного файла
process_file() {
    local file="$1"
    local file_num="$2"
    local total_files="$3"
    
    # Пропускаем файлы из списка исключений
    for exclude in "${EXCLUDE_FILES[@]}"; do
        if [[ "$file" == "./$exclude" ]]; then
            return
        fi
    done
    
    # Проверяем, является ли файл текстовым
    if ! is_text_file "$file"; then
        printf "${YELLOW}[%d/%d] Пропускаем бинарный файл: %s${NC}\n" "$file_num" "$total_files" "$file"
        return
    fi
    
    printf "${GREEN}[%d/%d] Обрабатываем: %s${NC}\n" "$file_num" "$total_files" "$file"
    
    # Удаляем начальный ./ из пути
    local clean_path="${file#./}"
    
    echo "" >> "$OUTPUT_FILE"
    echo "================================================================================" >> "$OUTPUT_FILE"
    echo "FILE: $clean_path" >> "$OUTPUT_FILE"
    echo "================================================================================" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
    
    # Проверяем размер файла
    local file_size=$(wc -c < "$file" 2>/dev/null || echo 0)
    local max_size=$((1024 * 1024)) # 1MB
    
    if [[ $file_size -gt $max_size ]]; then
        echo "[Файл слишком большой: $(($file_size / 1024))KB, показаны первые 1000 строк]" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        head -n 1000 "$file" >> "$OUTPUT_FILE" 2>/dev/null || echo "[Ошибка чтения файла]" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"
        echo "[... остальное содержимое обрезано ...]" >> "$OUTPUT_FILE"
    else
        cat "$file" >> "$OUTPUT_FILE" 2>/dev/null || echo "[Ошибка чтения файла]" >> "$OUTPUT_FILE"
    fi
    
    echo "" >> "$OUTPUT_FILE"
}

# Основная функция
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}    Сборщик проекта в единый файл      ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    
    # Проверяем, что мы в git репозитории
    if [[ ! -d .git ]] && [[ ! -f .git ]]; then
        echo -e "${YELLOW}Предупреждение: Текущая директория не является git репозиторием${NC}"
        echo "Продолжить? (y/n)"
        read -r response
        if [[ ! "$response" =~ ^[Yy]$ ]]; then
            echo "Отменено."
            exit 0
        fi
    fi
    
    # Удаляем старый выходной файл, если существует
    if [[ -f "$OUTPUT_FILE" ]]; then
        echo -e "${YELLOW}Файл $OUTPUT_FILE уже существует. Перезаписать? (y/n)${NC}"
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            rm "$OUTPUT_FILE"
        else
            echo "Отменено."
            exit 0
        fi
    fi
    
    echo -e "${BLUE}Анализ структуры проекта...${NC}"
    
    # Создаем заголовок файла
    {
        echo "================================================================================"
        echo "СНИМОК ПРОЕКТА"
        echo "Дата создания: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "Директория: $(pwd)"
        if git rev-parse --git-dir > /dev/null 2>&1; then
            echo "Git ветка: $(git branch --show-current 2>/dev/null || echo 'не определена')"
            echo "Git коммит: $(git rev-parse HEAD 2>/dev/null || echo 'не определен')"
        fi
        echo "================================================================================"
        echo ""
    } > "$OUTPUT_FILE"
    
    # Генерируем дерево директорий
    generate_tree >> "$OUTPUT_FILE"
    
    echo -e "${BLUE}Сбор файлов проекта...${NC}"
    
    # Создаем список файлов с учетом исключений
    excludes=$(create_find_excludes)
    eval "find . $excludes -type f -print" | sort > "$TEMP_FILE_LIST"
    
    # Подсчитываем количество текстовых файлов
    total_files=$(count_files)
    echo -e "${BLUE}Найдено текстовых файлов для обработки: $total_files${NC}"
    echo ""
    
    # Добавляем секцию с содержимым файлов
    {
        echo ""
        echo "================================================================================"
        echo "СОДЕРЖИМОЕ ФАЙЛОВ"
        echo "================================================================================"
    } >> "$OUTPUT_FILE"
    
    # Обрабатываем каждый файл
    file_num=0
    while IFS= read -r file; do
        if [[ -f "$file" ]] && is_text_file "$file"; then
            ((file_num++))
            process_file "$file" "$file_num" "$total_files"
        fi
    done < "$TEMP_FILE_LIST"
    
    # Удаляем временный файл
    rm -f "$TEMP_FILE_LIST"
    
    # Добавляем футер
    {
        echo ""
        echo "================================================================================"
        echo "КОНЕЦ СНИМКА ПРОЕКТА"
        echo "Обработано файлов: $file_num"
        echo "Дата завершения: $(date '+%Y-%m-%d %H:%M:%S')"
        echo "================================================================================"
    } >> "$OUTPUT_FILE"
    
    # Выводим статистику
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Готово!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Результат сохранен в: $OUTPUT_FILE"
    echo "Размер файла: $(du -h "$OUTPUT_FILE" | cut -f1)"
    echo "Обработано файлов: $file_num"
    echo ""
    
    # Предлагаем открыть файл
    echo "Открыть файл? (y/n)"
    read -r response
    if [[ "$response" =~ ^[Yy]$ ]]; then
        if command -v code >/dev/null 2>&1; then
            code "$OUTPUT_FILE"
        elif command -v open >/dev/null 2>&1; then
            open "$OUTPUT_FILE"
        else
            less "$OUTPUT_FILE"
        fi
    fi
}

# Запуск основной функции
main