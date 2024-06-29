O programa é uma implementação de um mutex distribuído. Para utilizar o programa, é recomendado utilizar a IDE Goland, pois junto com o programa, é possível utilizar configurações de execução já prontas para executar o programa em três processos diferentes. Isso pode ser acesso facilmente através da seção "Services"

A implementação utiliza canais unidirecionais para garantir que um canal de send é apenas utilizado para envio e não para recebimento de mensagens dentro da implementação

A implementação utiliza o pacote PerfectP2PLink para realizar a comunicação entre os processos por meio de comunicação ponto a ponto com TCP.

A implementação utiliza branded types para aumentar a legibilidade do código.

Para ativar as mensagens de debug, basta habilitar as flags de isInDebugMode nas implementações de mutex distribuídos e PerfectP2PLink

Após a implementação do trabalho, foi possível afirmar que a exclusão mútua é sempre garantida, visto que os padrões escritos no arquivo mxOUT.txt são sempre consistentes.

A exclusão mútua é um algoritmo simples e confiável, que se torna mais complexo devido à necessidade de garantir a comunicação consistente entre os processos, com um formato de mensagem que ambos os processos entendem.