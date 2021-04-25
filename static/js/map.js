let center = [55.641215, 37.615812]
// Функция ymaps.ready() будет вызвана, когда
// загрузятся все компоненты API, а также когда будет готово DOM-дерево.
ymaps.ready(init);
function init(){
    // Создание карты.
    var map = new ymaps.Map("map", {
        // Координаты центра карты.
        // Порядок по умолчанию: «широта, долгота».
        // Чтобы не определять координаты центра карты вручную,
        // воспользуйтесь инструментом Определение координат.
        center: center,
        // Уровень масштабирования. Допустимые значения:
        // от 0 (весь мир) до 19.
        zoom: 15
    });

    // map.controls.remove('zoomControl');
    map.controls.remove('searchControl');
    map.behaviors.disable('scrollZoom');
    if (window.innerWidth <= 576) map.behaviors.disable('drag');

    var marker = new ymaps.Placemark(center);
    map.geoObjects.add(marker);
}