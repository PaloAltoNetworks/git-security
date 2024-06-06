export default function (daysAgo: number) {
    var d = new Date();
    d.setDate(d.getDate() + daysAgo);
    return Math.ceil(d.getTime() / 1000);
}
