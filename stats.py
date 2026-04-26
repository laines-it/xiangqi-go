import json
import matplotlib.pyplot as plt
import numpy as np

def load_data(filename='three_searches_results.json'):
    with open(filename, 'r') as f:
        data = json.load(f)
    return data

def plot_cumulative_comparison(data):
    functions = data['functions']
    n = data['n']
    
    plt.figure(figsize=(14, 8))
    
    colors = [
    (0.8, 0.2, 0.1),   # 0 - темно-красный (теплый)
    (0.1, 0.3, 0.7),   # 1 - темно-синий (холодный)
    (0.9, 0.5, 0.1),   # 2 - оранжевый (теплый, светлее)
    (0.2, 0.6, 0.8),   # 3 - голубой (холодный, светлее)
    (1.0, 0.7, 0.2),   # 4 - золотистый (теплый, еще светлее)
    (0.5, 0.8, 0.9),   # 5 - бледно-голубой (холодный, светлее)
    (1.0, 0.9, 0.5),   # 6 - светло-желтый (теплый, очень светлый)
    (0.8, 0.9, 1.0)    # 7 - почти белый с синим отливом (холодный, самый светлый)
    ]   
    markers = ['o', 's', 'o', 's', 'o', 's', 'o', 's']
    
    for idx, func in enumerate(functions):
        name = func['name']
        cumulative = func['cumulative_sec']
        call_numbers = list(range(1, len(cumulative) + 1))
        
        plt.plot(call_numbers, cumulative, 
                marker=markers[idx % len(markers)], linewidth=2, markersize=6,
                color=colors[idx % len(colors)], label=f"{name} (всего: {cumulative[-1]:.4f} с)")
        
        # Добавляем значения на точки (каждую 2-ю точку для уменьшения загромождения)
        for i, (x, y) in enumerate(zip(call_numbers, cumulative)):
            if i % 2 == 0 or i == len(cumulative) - 1:  # каждую вторую и последнюю
                plt.annotate(f'{y:.3f}', (x, y), textcoords="offset points", 
                            xytext=(0, 8), ha='center', fontsize=7, 
                            color=colors[idx % len(colors)])
    
    plt.xlabel('Номер вызова', fontsize=12)
    plt.ylabel('Накопленное время (секунды)', fontsize=12)
    plt.title('Сравнение накопленных сумм времени выполнения для 8 функций\n(каждая точка = сумма предыдущих вызовов + текущий)', 
              fontsize=14)
    plt.grid(True, alpha=0.3)
    plt.legend(loc='upper left', fontsize=8, ncol=2)
    plt.xticks(range(1, n+1))
    
    # Автоматическая настройка пределов
    all_cumulative = []
    for func in functions:
        all_cumulative.extend(func['cumulative_sec'])
    
    if all_cumulative:
        max_val = max(all_cumulative)
        if max_val > 0:
            plt.ylim(0, max_val * 1.1)
        else:
            plt.ylim(0, 1)
    
    plt.tight_layout()
    plt.savefig('cumulative_comparison_8func.png', dpi=300, bbox_inches='tight')
    plt.show()

def plot_differences(data):
    """
    Строит график разностей между каждой парой функций для 8 функций (4 пары).
    Пары: (0,1), (2,3), (4,5), (6,7)
    """
    functions = data['functions']
    total_secs = [func['total_sec'] for func in functions]
    names = [func['name'] for func in functions]
    
    # Формируем пары: (0,1), (2,3), (4,5), (6,7) для 8 функций
    pairs = []
    for i in range(0, len(functions), 2):
        if i + 1 < len(functions):
            pairs.append((i, i + 1))
    
    if not pairs:
        print("Недостаточно функций для формирования пар (нужно минимум 2 функции)")
        return
    
    # Вычисляем разности для каждой пары
    differences = []
    abs_differences = []
    pair_names = []
    pair_labels = []
    
    print("\n" + "="*70)
    print("РАЗНОСТИ МЕЖДУ ПАРАМИ ФУНКЦИЙ")
    print("="*70)
    
    for idx, (i, j) in enumerate(pairs):
        diff = total_secs[i] - total_secs[j]
        abs_diff = abs(diff)
        differences.append(diff)
        abs_differences.append(abs_diff)
        pair_names.append(f"Пара {idx+1}")
        pair_labels.append(f"{names[i]}\nvs\n{names[j]}")
        
        print(f"\nПара {idx+1}:")
        print(f"  {names[i]}: {total_secs[i]:.6f} с")
        print(f"  {names[j]}: {total_secs[j]:.6f} с")
        print(f"  Разность ({names[i]} - {names[j]}): {diff:.6f} с")
        if diff > 0:
            print(f"  → {names[i]} медленнее на {abs_diff:.6f} с")
        elif diff < 0:
            print(f"  → {names[j]} медленнее на {abs_diff:.6f} с")
        else:
            print(f"  → Время выполнения одинаковое")
    
    # Создаём график разностей
    plt.figure(figsize=(14, 8))
    
    # График 3: Сравнение абсолютных значений (накопленная сумма для каждой пары)
    x = np.arange(len(pairs))
    width = 0.35
    
    first_vals = [total_secs[i] for i, j in pairs]
    second_vals = [total_secs[j] for i, j in pairs]
    
    bars3_1 = plt.bar(x - width/2, first_vals, width, label='Первая функция в паре', 
                      color='skyblue', alpha=0.7, edgecolor='black')
    bars3_2 = plt.bar(x + width/2, second_vals, width, label='Вторая функция в паре', 
                      color='lightcoral', alpha=0.7, edgecolor='black')
    
    # Добавляем значения
    for bar in bars3_1:
        height = bar.get_height()
        plt.annotate(f'{height:.3f}',
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),
                    textcoords="offset points",
                    ha='center', va='bottom', fontsize=9)
    
    for bar in bars3_2:
        height = bar.get_height()
        plt.annotate(f'{height:.3f}',
                    xy=(bar.get_x() + bar.get_width() / 2, height),
                    xytext=(0, 3),
                    textcoords="offset points",
                    ha='center', va='bottom', fontsize=9)
    
    plt.xlabel('Пары функций', fontsize=12)
    plt.ylabel('Суммарное время (секунды)', fontsize=12)
    plt.title('Сравнение суммарного времени в каждой паре', fontsize=12)
    plt.xticks(x, pair_labels, fontsize=9)
    plt.legend(fontsize=10)
    plt.grid(True, alpha=0.3, axis='y')
    
    plt.suptitle('Анализ разностей для 8 функций (4 пары)', fontsize=16, fontweight='bold')
    plt.tight_layout()
    plt.savefig('differences_plot_8func.png', dpi=300, bbox_inches='tight')
    plt.show()
    
    # Вывод сводной статистики
    print("\n" + "="*70)
    print("СВОДНАЯ СТАТИСТИКА ПО ВСЕМ ПАРАМ")
    print("="*70)
    print(f"\n{'Пара':<10} {'Разность (с)':<15} {'|Разность| (с)':<15} {'Отн. разность (%)':<20}")
    print("-" * 70)
    for idx in range(len(pairs)):
        print(f"Пара {idx+1:<4} {differences[idx]:+15.6f} {abs_differences[idx]:15.6f} {percentages[idx]:17.2f}")
    
    print(f"\nСредняя абсолютная разность: {np.mean(abs_differences):.6f} с")
    print(f"Максимальная абсолютная разность: {max(abs_differences):.6f} с")
    print(f"Минимальная абсолютная разность: {min(abs_differences):.6f} с")
    print("="*70)

def print_statistics(data):
    print("\n" + "="*70)
    print("СТАТИСТИКА ДЛЯ 8 ФУНКЦИЙ")
    print("="*70)
    print(f"Количество вызовов на функцию: {data['n']}")
    print(f"Параметр depth: {data['depth']}\n")
    
    # Сортируем функции по суммарному времени
    functions_sorted = sorted(data['functions'], key=lambda x: x['total_sec'])
    
    print(f"{'Функция':<25} {'Суммарное (с)':<15} {'Среднее (с)':<15} {'Мин/Макс (с)':<20}")
    print("-" * 75)
    
    for func in functions_sorted:
        durations = [r['duration_sec'] for r in func['results']]
        print(f"{func['name']:<25} {func['total_sec']:>12.6f}   {func['average_sec']:>11.6f}   "
              f"{min(durations):.4f} / {max(durations):.4f}")
    
    # Самая быстрая и самая медленная функция
    fastest = functions_sorted[0]
    slowest = functions_sorted[-1]
    print(f"\n🏆 Самая быстрая: {fastest['name']} ({fastest['total_sec']:.6f} с)")
    print(f"🐌 Самая медленная: {slowest['name']} ({slowest['total_sec']:.6f} с)")
    print(f"📊 Разница: {slowest['total_sec'] - fastest['total_sec']:.6f} с "
          f"(в {slowest['total_sec']/fastest['total_sec']:.2f} раз)")
    print("="*70)

def main():
    # Загружаем данные
    data = load_data('three_searches_results.json')
    
    # Выводим статистику
    print_statistics(data)
    
    # Строим график накопленных сумм
    plot_cumulative_comparison(data)
    
    # Строим график разностей между парами
    plot_differences(data)

if __name__ == "__main__":
    main()